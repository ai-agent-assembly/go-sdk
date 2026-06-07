//! Go C-ABI static library bindings for Agent Assembly.
//!
//! Thin C-ABI shim over [`aa_sdk_client`]: every entry point delegates to the
//! shared [`AssemblyClient`], so the Unix-domain-socket transport, IPC wire
//! codec, session lifecycle, and advisory preflight live in exactly one place
//! (mirroring the napi/pyo3 shims, AAASM-2552 / AAASM-2703). This crate adds
//! only C-ABI marshalling, opaque-handle management, and panic/null guards — it
//! has **no transport or policy logic of its own**.
//!
//! # Trust model
//!
//! The SDK is **not** a security boundary. Authoritative credential scanning,
//! redaction, and normalization happen at the runtime chokepoint (`aa-runtime`,
//! AAASM-2568), which re-scans every event unconditionally. Policy decisions are
//! server-side; this binding deliberately exposes **no** `query_policy` surface.

use core::ffi::c_char;
use std::ffi::{CStr, CString};
use std::panic::{catch_unwind, AssertUnwindSafe};
use std::path::PathBuf;

use aa_sdk_client::{ipc::spawn_ipc_thread, AssemblyClient, SdkClientError};

/// C-ABI status code returned by every fallible entry point.
pub type AaStatus = i32;

pub const AA_STATUS_OK: AaStatus = 0;
pub const AA_STATUS_NULL_POINTER: AaStatus = 1;
pub const AA_STATUS_INVALID_UTF8: AaStatus = 2;
/// The client has been shut down (or its background IPC thread has exited).
pub const AA_STATUS_NOT_CONNECTED: AaStatus = 3;
pub const AA_STATUS_MUTEX_POISONED: AaStatus = 4;
/// The background IPC thread could not be spawned.
pub const AA_STATUS_IPC_ERROR: AaStatus = 5;
/// The IPC command channel is closed, so the event could not be enqueued.
pub const AA_STATUS_CHANNEL_CLOSED: AaStatus = 6;
/// A panic was caught at the FFI boundary (never propagated across it).
pub const AA_STATUS_PANIC: AaStatus = 7;

/// Opaque handle to an active Agent Assembly session.
///
/// Wraps a shared [`AssemblyClient`]. Created by [`aa_connect`]; released by
/// [`aa_disconnect`]. Opaque to C — go-sdk holds it as a `*aa_client_handle`.
#[allow(non_camel_case_types)]
pub struct aa_client_handle {
    client: AssemblyClient,
}

/// Map a shared-client error onto a stable C-ABI status code.
fn status_for(err: &SdkClientError) -> AaStatus {
    match err {
        SdkClientError::Shutdown => AA_STATUS_NOT_CONNECTED,
        SdkClientError::LockPoisoned => AA_STATUS_MUTEX_POISONED,
        SdkClientError::ChannelClosed => AA_STATUS_CHANNEL_CLOSED,
    }
}

/// Open a session to the runtime over the Unix socket at `endpoint`.
///
/// On success, `*out_client` receives an owned handle that must be released
/// with [`aa_disconnect`].
///
/// # Safety
///
/// `endpoint` must be a valid NUL-terminated C string; `out_client` must be a
/// valid, writable pointer to a `*mut aa_client_handle`.
#[no_mangle]
pub unsafe extern "C" fn aa_connect(
    endpoint: *const c_char,
    out_client: *mut *mut aa_client_handle,
) -> AaStatus {
    catch_unwind(AssertUnwindSafe(|| {
        if endpoint.is_null() || out_client.is_null() {
            return AA_STATUS_NULL_POINTER;
        }

        // SAFETY: `endpoint` null-checked above.
        let endpoint = match unsafe { CStr::from_ptr(endpoint) }.to_str() {
            Ok(value) => value,
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };

        // All transport lives in aa-sdk-client; we only spawn + own the client.
        let ipc = match spawn_ipc_thread(PathBuf::from(endpoint)) {
            Ok(handle) => handle,
            Err(_) => return AA_STATUS_IPC_ERROR,
        };

        let handle = Box::new(aa_client_handle {
            client: AssemblyClient::new(ipc, Vec::new()),
        });

        // SAFETY: `out_client` null-checked above.
        unsafe {
            *out_client = Box::into_raw(handle);
        }

        AA_STATUS_OK
    }))
    .unwrap_or(AA_STATUS_PANIC)
}

/// Report an audit event `(event_type, details)` to the runtime.
///
/// Delegates to [`AssemblyClient::report_event`]; `details` passes through the
/// advisory preflight before shipping, and the runtime re-scans regardless.
///
/// # Safety
///
/// `client` must be a handle from [`aa_connect`] that has not been
/// disconnected; `event_type` and `details` must be valid NUL-terminated C
/// strings.
#[no_mangle]
pub unsafe extern "C" fn aa_send_event(
    client: *mut aa_client_handle,
    event_type: *const c_char,
    details: *const c_char,
) -> AaStatus {
    catch_unwind(AssertUnwindSafe(|| {
        if client.is_null() || event_type.is_null() || details.is_null() {
            return AA_STATUS_NULL_POINTER;
        }

        // SAFETY: pointers null-checked above.
        let event_type = match unsafe { CStr::from_ptr(event_type) }.to_str() {
            Ok(value) => value.to_owned(),
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };
        // SAFETY: pointers null-checked above.
        let details = match unsafe { CStr::from_ptr(details) }.to_str() {
            Ok(value) => value.to_owned(),
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };

        // SAFETY: `client` null-checked above; `&*` borrows the boxed handle.
        let handle = unsafe { &*client };
        match handle.client.report_event(event_type, details) {
            Ok(()) => AA_STATUS_OK,
            Err(err) => status_for(&err),
        }
    }))
    .unwrap_or(AA_STATUS_PANIC)
}

/// Shut down the session and free the handle.
///
/// Blocks on the background IPC thread's join (the shared client signals it).
/// After this call the handle is freed and must not be reused.
///
/// # Safety
///
/// `client` must be a handle previously returned by [`aa_connect`] and not yet
/// disconnected.
#[no_mangle]
pub unsafe extern "C" fn aa_disconnect(client: *mut aa_client_handle) -> AaStatus {
    catch_unwind(AssertUnwindSafe(|| {
        if client.is_null() {
            return AA_STATUS_NULL_POINTER;
        }

        // SAFETY: `client` originated from `Box::into_raw` in `aa_connect`;
        // reclaiming it here frees the handle (and joins the IPC thread via
        // `shutdown`).
        let handle = unsafe { Box::from_raw(client) };
        match handle.client.shutdown() {
            Ok(()) => AA_STATUS_OK,
            Err(err) => status_for(&err),
        }
    }))
    .unwrap_or(AA_STATUS_PANIC)
}

/// Free a C string previously handed out by this crate.
///
/// Retained as a stable ABI helper for C callers; the event API no longer
/// returns owned strings.
///
/// # Safety
///
/// `value` must be a pointer previously returned by this crate, or null.
#[no_mangle]
pub unsafe extern "C" fn aa_free_string(value: *mut c_char) {
    let _ = catch_unwind(AssertUnwindSafe(|| {
        if value.is_null() {
            return;
        }
        // SAFETY: `value` originated from `CString::into_raw` in this crate.
        unsafe {
            drop(CString::from_raw(value));
        }
    }));
}

/// Free a byte buffer previously handed out by this crate.
///
/// # Safety
///
/// `bytes`/`len` must originate from an allocation owned by this crate, or
/// `bytes` may be null.
#[no_mangle]
pub unsafe extern "C" fn aa_free_bytes(bytes: *mut u8, len: usize) {
    let _ = catch_unwind(AssertUnwindSafe(|| {
        if bytes.is_null() {
            return;
        }
        // SAFETY: caller guarantees pointer/length originate from this crate.
        unsafe {
            drop(Vec::from_raw_parts(bytes, len, len));
        }
    }));
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ptr;

    /// connect → disconnect round-trip against a (deliberately absent) socket.
    /// `spawn_ipc_thread` returns a handle immediately; the background thread
    /// exits when it cannot reach the socket, so `disconnect` joins promptly.
    #[test]
    fn connect_then_disconnect_roundtrip() {
        let endpoint = CString::new("/tmp/aa-ffi-go-nonexistent.sock").expect("valid endpoint");

        let mut client: *mut aa_client_handle = ptr::null_mut();
        // SAFETY: valid pointers from controlled test context.
        let connect = unsafe { aa_connect(endpoint.as_ptr(), &mut client) };
        assert_eq!(connect, AA_STATUS_OK);
        assert!(!client.is_null());

        // SAFETY: handle returned by `aa_connect`.
        let disconnect = unsafe { aa_disconnect(client) };
        assert_eq!(disconnect, AA_STATUS_OK);
    }

    #[test]
    fn connect_rejects_null_out_pointer() {
        let endpoint = CString::new("/tmp/aa.sock").expect("valid endpoint");
        // SAFETY: deliberate null out-pointer to validate the guard.
        let status = unsafe { aa_connect(endpoint.as_ptr(), ptr::null_mut()) };
        assert_eq!(status, AA_STATUS_NULL_POINTER);
    }

    #[test]
    fn connect_rejects_invalid_utf8() {
        let bytes = [0xFF, 0x00];
        // SAFETY: test-only pointer with an invalid-UTF-8, NUL-terminated payload.
        let invalid = bytes.as_ptr().cast::<c_char>();
        let mut client: *mut aa_client_handle = ptr::null_mut();
        // SAFETY: deliberate invalid UTF-8 input for status-mapping validation.
        let status = unsafe { aa_connect(invalid, &mut client) };
        assert_eq!(status, AA_STATUS_INVALID_UTF8);
        assert!(client.is_null());
    }

    #[test]
    fn send_event_rejects_null_client() {
        let event_type = CString::new("tool_call").expect("valid");
        let details = CString::new("searched for cats").expect("valid");
        // SAFETY: deliberate null client to validate the guard.
        let status =
            unsafe { aa_send_event(ptr::null_mut(), event_type.as_ptr(), details.as_ptr()) };
        assert_eq!(status, AA_STATUS_NULL_POINTER);
    }

    #[test]
    fn disconnect_rejects_null_client() {
        // SAFETY: deliberate null client to validate the guard.
        let status = unsafe { aa_disconnect(ptr::null_mut()) };
        assert_eq!(status, AA_STATUS_NULL_POINTER);
    }

    #[test]
    fn free_helpers_tolerate_null_and_owned() {
        // SAFETY: null is explicitly tolerated.
        unsafe { aa_free_string(ptr::null_mut()) };
        // SAFETY: null is explicitly tolerated.
        unsafe { aa_free_bytes(ptr::null_mut(), 0) };

        let owned = CString::new("owned").expect("valid");
        let raw = owned.into_raw();
        // SAFETY: pointer came from CString::into_raw above.
        unsafe { aa_free_string(raw) };
    }
}

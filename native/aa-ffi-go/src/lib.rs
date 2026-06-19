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
//! server-side. [`aa_query_policy`] exposes the runtime's synchronous policy
//! query, but it is **advisory and fail-open**: when the runtime is unreachable
//! or slow it returns `ALLOW` rather than blocking, because the runtime / proxy
//! / eBPF layers are the authoritative enforcement points (AAASM-3021).

use core::ffi::c_char;
use std::ffi::{CStr, CString};
use std::panic::{catch_unwind, AssertUnwindSafe};
use std::path::PathBuf;

use aa_proto::assembly::common::v1::{ActionType, AgentId, Decision};
use aa_proto::assembly::policy::v1::{
    action_context, ActionContext, CheckActionRequest, LlmCallContext, ToolCallContext,
};
use aa_sdk_client::{ipc::spawn_ipc_thread, AssemblyClient, AssemblyConfig, SdkClientError};

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
/// The gateway gRPC endpoint could not be reached for registration. Unlike a
/// policy query, [`aa_register`] is **fail-closed** — it surfaces this rather
/// than silently proceeding without a credential token.
pub const AA_STATUS_GATEWAY_UNREACHABLE: AaStatus = 8;
/// The gateway was reached but rejected the `Register` call (e.g. an invalid
/// `did:key`). [`aa_register`] surfaces it; registration never fails open.
pub const AA_STATUS_REGISTER_FAILED: AaStatus = 9;

/// C-ABI policy decision returned by [`aa_query_policy`].
///
/// Mirrors `aa_proto::assembly::common::v1::Decision`. `UNSPECIFIED` is folded
/// onto `ALLOW` so an unset/garbled decision never silently blocks.
pub type AaDecision = i32;

/// Action permitted. Also returned when the query fails open (see
/// [`aa_query_policy`]) or the runtime returns an unspecified decision.
pub const AA_DECISION_ALLOW: AaDecision = 0;
/// Action blocked.
pub const AA_DECISION_DENY: AaDecision = 1;
/// Action held for human approval.
pub const AA_DECISION_PENDING: AaDecision = 2;
/// Action permitted but sensitive fields must be redacted first.
pub const AA_DECISION_REDACT: AaDecision = 3;

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
        // The event paths never trigger a synchronous query, so this is
        // unreachable for them; map it conservatively to "not connected".
        // [`aa_query_policy`] handles `QueryFailed` itself by failing open.
        SdkClientError::QueryFailed => AA_STATUS_NOT_CONNECTED,
        // Registration-only failures (see [`aa_register`]), surfaced verbatim
        // so the caller can distinguish "gateway down" from "gateway said no".
        SdkClientError::GatewayUnreachable => AA_STATUS_GATEWAY_UNREACHABLE,
        SdkClientError::RegisterFailed(_) => AA_STATUS_REGISTER_FAILED,
    }
}

/// Parse the C-supplied action-type string onto the proto [`ActionType`].
///
/// Accepts the snake_case proto names (and a couple of common aliases).
/// Unknown values fall back to [`ActionType::ToolCall`] — the broadest action
/// category — so a typo never silently skips policy evaluation.
fn parse_action_type(raw: &str) -> ActionType {
    match raw {
        "llm_call" | "llm" => ActionType::LlmCall,
        "tool_call" | "tool" => ActionType::ToolCall,
        "file_operation" | "file_op" | "file" => ActionType::FileOperation,
        "network_call" | "network" => ActionType::NetworkCall,
        "process_exec" | "process" => ActionType::ProcessExec,
        "agent_spawn" => ActionType::AgentSpawn,
        "tool_result" => ActionType::ToolResult,
        _ => ActionType::ToolCall,
    }
}

/// Map a proto [`Decision`] code onto the stable C-ABI [`AaDecision`].
///
/// Any value that is not a recognized non-allow decision (including
/// `UNSPECIFIED`) folds onto [`AA_DECISION_ALLOW`] so the binding never blocks
/// on a decision it cannot interpret.
fn decision_for(decision: i32) -> AaDecision {
    match Decision::try_from(decision) {
        Ok(Decision::Deny) => AA_DECISION_DENY,
        Ok(Decision::Pending) => AA_DECISION_PENDING,
        Ok(Decision::Redact) => AA_DECISION_REDACT,
        Ok(Decision::Allow) | Ok(Decision::Unspecified) | Err(_) => AA_DECISION_ALLOW,
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

/// Register this agent with the governance gateway and store the issued
/// credential token on the session.
///
/// Delegates to [`AssemblyClient::register`] — the SDK's only direct gateway
/// gRPC call (ADR 0004). It hits `AgentLifecycleService.Register`; the returned
/// `credential_token` is held by the shared client and attached to every later
/// [`aa_query_policy`] so the gateway's `validate_credential_token` does not
/// deny a registered agent. On success `*out_policy_id` receives the
/// gateway-assigned policy id as an owned, NUL-terminated string the caller must
/// release with [`aa_free_string`].
///
/// # Fail-closed
///
/// Unlike [`aa_query_policy`], registration is **not** advisory: a failure
/// surfaces rather than failing open. `AA_STATUS_GATEWAY_UNREACHABLE` means the
/// gateway gRPC endpoint could not be reached; `AA_STATUS_REGISTER_FAILED` means
/// the gateway rejected the call (e.g. an invalid `did:key`). `*out_policy_id`
/// is left untouched on any non-`AA_STATUS_OK` return.
///
/// # Arguments
///
/// * `agent_id` — the agent identity to register (derived into a `did:key` +
///   Ed25519 public key by the shared client).
/// * `name` / `framework` — descriptive metadata the gateway records.
/// * `gateway_endpoint` — the gRPC endpoint (e.g. `"http://127.0.0.1:50051"`);
///   may be null to let the shared client resolve it from `AA_GATEWAY_ENDPOINT`
///   or its default.
/// * `team_id` / `parent_agent_id` — the agent's lineage/team scoping forwarded
///   to the gateway on the native register (AAASM-3415, mirroring the pyo3/napi
///   shims): `team_id` drives team-budget attribution and `parent_agent_id` the
///   topology graph. Both are optional — null leaves the agent team-unscoped /
///   root.
///
/// # Safety
///
/// `client` must be a handle from [`aa_connect`] that has not been
/// disconnected. `agent_id`, `name`, and `framework` must be valid
/// NUL-terminated C strings. `gateway_endpoint`, `team_id`, and
/// `parent_agent_id` must each be a valid NUL-terminated C string or null.
/// `out_policy_id` must be a valid, writable pointer.
// `team_id` and `parent_agent_id` are distinct optional lineage fields crossing
// the C boundary; bundling them into a struct would force Go callers to build
// one, complicating the cgo call site for no benefit (mirrors the pyo3 shim).
#[allow(clippy::too_many_arguments)]
#[no_mangle]
pub unsafe extern "C" fn aa_register(
    client: *mut aa_client_handle,
    agent_id: *const c_char,
    name: *const c_char,
    framework: *const c_char,
    gateway_endpoint: *const c_char,
    team_id: *const c_char,
    parent_agent_id: *const c_char,
    out_policy_id: *mut *mut c_char,
) -> AaStatus {
    catch_unwind(AssertUnwindSafe(|| {
        if client.is_null()
            || agent_id.is_null()
            || name.is_null()
            || framework.is_null()
            || out_policy_id.is_null()
        {
            return AA_STATUS_NULL_POINTER;
        }

        // SAFETY: pointers null-checked above.
        let agent_id = match unsafe { CStr::from_ptr(agent_id) }.to_str() {
            Ok(value) => value.to_owned(),
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };
        // SAFETY: pointers null-checked above.
        let name = match unsafe { CStr::from_ptr(name) }.to_str() {
            Ok(value) => value.to_owned(),
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };
        // SAFETY: pointers null-checked above.
        let framework = match unsafe { CStr::from_ptr(framework) }.to_str() {
            Ok(value) => value.to_owned(),
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };
        // Optional: null ⇒ let the shared client resolve the endpoint.
        // SAFETY: null-checked inside the helper before dereferencing.
        let gateway_endpoint = match unsafe { optional_cstr(gateway_endpoint) } {
            Ok(value) => value,
            Err(status) => return status,
        };
        // Optional lineage: null ⇒ None (team-unscoped / root).
        // SAFETY: each is null-checked by the helper before dereferencing.
        let team_id = match unsafe { optional_cstr(team_id) } {
            Ok(value) => value,
            Err(status) => return status,
        };
        let parent_agent_id = match unsafe { optional_cstr(parent_agent_id) } {
            Ok(value) => value,
            Err(status) => return status,
        };

        let config = AssemblyConfig {
            agent_id,
            // Registration is a gateway gRPC call; the runtime UDS socket is not
            // consulted here, so leave the path to default resolution.
            socket_path: None,
            gateway_endpoint,
            team_id,
            parent_agent_id,
        };

        // `register` is async (tonic). Drive the one future to completion on a
        // private current-thread runtime — the C-ABI surface is synchronous.
        let runtime = match tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
        {
            Ok(rt) => rt,
            Err(_) => return AA_STATUS_IPC_ERROR,
        };

        // SAFETY: `client` null-checked above; `&*` borrows the boxed handle.
        let handle = unsafe { &*client };
        let policy_id = match runtime.block_on(handle.client.register(&config, name, framework)) {
            Ok(policy_id) => policy_id,
            // Fail-closed: surface the failure instead of proceeding tokenless.
            Err(err) => return status_for(&err),
        };

        // A NUL inside the policy id would truncate it; fall back to empty
        // rather than fail a successful registration over a gateway-controlled
        // string.
        let policy_c = CString::new(policy_id).unwrap_or_default();

        // SAFETY: out-pointer null-checked above.
        unsafe {
            *out_policy_id = policy_c.into_raw();
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

/// Synchronously query the runtime for a policy decision on an action.
///
/// Builds a `CheckActionRequest` from the supplied inputs, sends it to the
/// runtime over the shared session, and blocks (up to the shared client's 5 s
/// timeout) for the decision. Delegates entirely to
/// [`AssemblyClient::query_policy`]; this shim only marshals across the C
/// boundary.
///
/// On success, `*out_decision` receives an [`AaDecision`] and `*out_reason`
/// receives an owned, NUL-terminated reason string the caller must release with
/// [`aa_free_string`] (always non-null on `AA_STATUS_OK`, even when empty).
///
/// # Fail-open
///
/// The SDK is **advisory, not authoritative**. If the runtime fails to return a
/// decision — a timeout, an unreachable runtime, or a closed session — this
/// returns `AA_STATUS_OK` with `*out_decision = AA_DECISION_ALLOW` and a reason
/// explaining the fail-open, since an unreachable or slow runtime must never
/// block the agent (the runtime / proxy / eBPF layers enforce authoritatively).
/// Only an internal poisoned lock surfaces as `AA_STATUS_MUTEX_POISONED`.
///
/// # Arguments
///
/// * `agent_id` — the calling agent's own id (the `agent_id.agent_id` field).
/// * `action_type` — snake_case proto action name (e.g. `"tool_call"`,
///   `"llm_call"`); unknown values fall back to `tool_call`.
/// * `tool_name` — registered tool name; may be null. Used for `tool_call` /
///   `tool_result` actions and as the model for `llm_call`.
/// * `args_json` — JSON-encoded argument map; may be null. Carried as the tool
///   `args_json` bytes for tool actions.
///
/// # Safety
///
/// `client` must be a handle from [`aa_connect`] that has not been
/// disconnected. `agent_id` and `action_type` must be valid NUL-terminated C
/// strings. `tool_name` and `args_json` must each be a valid NUL-terminated C
/// string or null. `out_decision` and `out_reason` must be valid, writable
/// pointers.
#[no_mangle]
pub unsafe extern "C" fn aa_query_policy(
    client: *mut aa_client_handle,
    agent_id: *const c_char,
    action_type: *const c_char,
    tool_name: *const c_char,
    args_json: *const c_char,
    out_decision: *mut AaDecision,
    out_reason: *mut *mut c_char,
) -> AaStatus {
    catch_unwind(AssertUnwindSafe(|| {
        if client.is_null()
            || agent_id.is_null()
            || action_type.is_null()
            || out_decision.is_null()
            || out_reason.is_null()
        {
            return AA_STATUS_NULL_POINTER;
        }

        // SAFETY: pointers null-checked above.
        let agent_id = match unsafe { CStr::from_ptr(agent_id) }.to_str() {
            Ok(value) => value.to_owned(),
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };
        // SAFETY: pointers null-checked above.
        let action_type = match unsafe { CStr::from_ptr(action_type) }.to_str() {
            Ok(value) => value.to_owned(),
            Err(_) => return AA_STATUS_INVALID_UTF8,
        };
        // Optional inputs: null ⇒ None. SAFETY: each is null-checked by the
        // helper before dereferencing.
        let tool_name = match unsafe { optional_cstr(tool_name) } {
            Ok(value) => value,
            Err(status) => return status,
        };
        let args_json = match unsafe { optional_cstr(args_json) } {
            Ok(value) => value,
            Err(status) => return status,
        };

        let action = parse_action_type(&action_type);
        let context = build_context(action, tool_name, args_json);

        let request = CheckActionRequest {
            agent_id: Some(AgentId {
                agent_id,
                ..Default::default()
            }),
            action_type: action.into(),
            context,
            ..Default::default()
        };

        // SAFETY: `client` null-checked above; `&*` borrows the boxed handle.
        let handle = unsafe { &*client };
        let (decision, reason) = match handle.client.query_policy(request) {
            Ok(resp) => (decision_for(resp.decision), resp.reason),
            // Fail-open: the SDK is advisory, so any failure to obtain a
            // decision from the runtime — a timeout (`QueryFailed`), an
            // unreachable runtime whose IPC thread exited (`ChannelClosed`), or
            // a shut-down session (`Shutdown`) — yields ALLOW rather than
            // blocking the agent. The runtime / proxy / eBPF layers enforce
            // authoritatively. Only a poisoned lock (an internal panic) is a
            // hard error, since it signals the binding itself is broken.
            Err(SdkClientError::LockPoisoned) => return AA_STATUS_MUTEX_POISONED,
            Err(_) => (
                AA_DECISION_ALLOW,
                "policy query failed; failing open (SDK is advisory)".to_owned(),
            ),
        };

        // A NUL inside the reason would truncate it; fall back to empty rather
        // than fail the whole query over a runtime-controlled string.
        let reason_c = CString::new(reason).unwrap_or_default();

        // SAFETY: out-pointers null-checked above.
        unsafe {
            *out_decision = decision;
            *out_reason = reason_c.into_raw();
        }

        AA_STATUS_OK
    }))
    .unwrap_or(AA_STATUS_PANIC)
}

/// Read an optional NUL-terminated C string: null ⇒ `Ok(None)`, valid UTF-8 ⇒
/// `Ok(Some(..))`, invalid UTF-8 ⇒ `Err(AA_STATUS_INVALID_UTF8)`.
///
/// # Safety
///
/// `ptr` must be null or a valid NUL-terminated C string.
unsafe fn optional_cstr(ptr: *const c_char) -> Result<Option<String>, AaStatus> {
    if ptr.is_null() {
        return Ok(None);
    }
    // SAFETY: `ptr` null-checked above.
    match unsafe { CStr::from_ptr(ptr) }.to_str() {
        Ok(value) => Ok(Some(value.to_owned())),
        Err(_) => Err(AA_STATUS_INVALID_UTF8),
    }
}

/// Build the typed [`ActionContext`] for `action` from the optional inputs.
///
/// Only `tool_call` / `tool_result` and `llm_call` carry context here; other
/// action types submit no context (the runtime still evaluates them).
fn build_context(
    action: ActionType,
    tool_name: Option<String>,
    args_json: Option<String>,
) -> Option<ActionContext> {
    match action {
        ActionType::ToolCall | ActionType::ToolResult => Some(ActionContext {
            action: Some(action_context::Action::ToolCall(ToolCallContext {
                tool_name: tool_name.unwrap_or_default(),
                args_json: args_json.unwrap_or_default().into_bytes(),
                ..Default::default()
            })),
        }),
        ActionType::LlmCall => Some(ActionContext {
            action: Some(action_context::Action::LlmCall(LlmCallContext {
                model: tool_name.unwrap_or_default(),
                ..Default::default()
            })),
        }),
        _ => None,
    }
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
    fn register_rejects_null_client() {
        let agent = CString::new("agent-1").expect("valid");
        let name = CString::new("My Agent").expect("valid");
        let framework = CString::new("langchain").expect("valid");
        let mut policy_id: *mut c_char = ptr::null_mut();
        // SAFETY: deliberate null client to validate the guard.
        let status = unsafe {
            aa_register(
                ptr::null_mut(),
                agent.as_ptr(),
                name.as_ptr(),
                framework.as_ptr(),
                ptr::null(),
                ptr::null(),
                ptr::null(),
                &mut policy_id,
            )
        };
        assert_eq!(status, AA_STATUS_NULL_POINTER);
        assert!(policy_id.is_null());
    }

    /// Registration is fail-closed: with no gateway listening at the supplied
    /// endpoint, `aa_register` must surface `AA_STATUS_GATEWAY_UNREACHABLE`
    /// rather than fail open like a policy query. `out_policy_id` stays null.
    #[test]
    fn register_fails_closed_when_gateway_unreachable() {
        // Connect a session (the runtime socket need not exist; register does
        // not consult it), then point register at a dead gateway endpoint.
        let endpoint = CString::new(format!(
            "/tmp/aa-ffi-go-register-{}.sock",
            std::process::id()
        ))
        .expect("valid endpoint");
        let mut client: *mut aa_client_handle = ptr::null_mut();
        // SAFETY: valid pointers from a controlled test context.
        assert_eq!(
            unsafe { aa_connect(endpoint.as_ptr(), &mut client) },
            AA_STATUS_OK
        );

        let agent = CString::new("agent-1").expect("valid");
        let name = CString::new("My Agent").expect("valid");
        let framework = CString::new("langchain").expect("valid");
        // Port 1 is never a live gateway, so the gRPC connect fails fast.
        let gateway = CString::new("http://127.0.0.1:1").expect("valid");
        let mut policy_id: *mut c_char = ptr::null_mut();
        // SAFETY: handle from aa_connect; valid in/out pointers.
        let status = unsafe {
            aa_register(
                client,
                agent.as_ptr(),
                name.as_ptr(),
                framework.as_ptr(),
                gateway.as_ptr(),
                ptr::null(),
                ptr::null(),
                &mut policy_id,
            )
        };

        assert_eq!(
            status, AA_STATUS_GATEWAY_UNREACHABLE,
            "registration must fail closed, not open, when the gateway is down"
        );
        assert!(
            policy_id.is_null(),
            "no policy id is written on a failed registration"
        );

        // SAFETY: handle from aa_connect, not yet disconnected.
        assert_eq!(unsafe { aa_disconnect(client) }, AA_STATUS_OK);
    }

    /// The lineage params (`team_id` / `parent_agent_id`) are accepted and
    /// marshalled into `AssemblyConfig` without disturbing the fail-closed
    /// contract: with both set and no gateway listening, `aa_register` still
    /// surfaces `AA_STATUS_GATEWAY_UNREACHABLE` and writes no policy id. This
    /// drives the new optional-cstr branches for non-null lineage (AAASM-3444).
    #[test]
    fn register_forwards_lineage_and_still_fails_closed() {
        let endpoint = CString::new(format!(
            "/tmp/aa-ffi-go-register-lineage-{}.sock",
            std::process::id()
        ))
        .expect("valid endpoint");
        let mut client: *mut aa_client_handle = ptr::null_mut();
        // SAFETY: valid pointers from a controlled test context.
        assert_eq!(
            unsafe { aa_connect(endpoint.as_ptr(), &mut client) },
            AA_STATUS_OK
        );

        let agent = CString::new("agent-1").expect("valid");
        let name = CString::new("My Agent").expect("valid");
        let framework = CString::new("langchain").expect("valid");
        let gateway = CString::new("http://127.0.0.1:1").expect("valid");
        let team = CString::new("team-platform").expect("valid");
        let parent = CString::new("agent-orchestrator").expect("valid");
        let mut policy_id: *mut c_char = ptr::null_mut();
        // SAFETY: handle from aa_connect; valid in/out pointers, non-null lineage.
        let status = unsafe {
            aa_register(
                client,
                agent.as_ptr(),
                name.as_ptr(),
                framework.as_ptr(),
                gateway.as_ptr(),
                team.as_ptr(),
                parent.as_ptr(),
                &mut policy_id,
            )
        };

        assert_eq!(
            status, AA_STATUS_GATEWAY_UNREACHABLE,
            "lineage params must not change the fail-closed contract"
        );
        assert!(policy_id.is_null());

        // SAFETY: handle from aa_connect, not yet disconnected.
        assert_eq!(unsafe { aa_disconnect(client) }, AA_STATUS_OK);
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

    #[test]
    fn query_policy_rejects_null_client() {
        let agent = CString::new("agent-1").expect("valid");
        let action = CString::new("tool_call").expect("valid");
        let mut decision: AaDecision = AA_DECISION_DENY;
        let mut reason: *mut c_char = ptr::null_mut();
        // SAFETY: deliberate null client to validate the guard.
        let status = unsafe {
            aa_query_policy(
                ptr::null_mut(),
                agent.as_ptr(),
                action.as_ptr(),
                ptr::null(),
                ptr::null(),
                &mut decision,
                &mut reason,
            )
        };
        assert_eq!(status, AA_STATUS_NULL_POINTER);
    }

    /// A mock UDS runtime that answers the policy query with a Deny
    /// `CheckActionResponse` drives the binding to return `AA_DECISION_DENY`.
    ///
    /// Mirrors `aa-sdk-client`'s `query_policy_returns_runtime_decision`: the
    /// mock reads the heartbeat + policy-query frames, then writes a Deny
    /// response. The FFI calls run on this thread (raw handles are not `Send`),
    /// while the mock server runs on a background thread with its own runtime.
    #[test]
    fn query_policy_returns_deny_from_runtime() {
        use std::io::{Read, Write};
        use std::os::unix::net::UnixListener;

        use aa_sdk_client::codec;
        use prost::Message;

        let socket_path = format!("/tmp/aa-ffi-go-query-deny-{}.sock", std::process::id());
        let _ = std::fs::remove_file(&socket_path);
        let listener = UnixListener::bind(&socket_path).expect("bind mock socket");

        // Mock runtime on a plain blocking thread: read the heartbeat, then the
        // policy-query frame, then reply with a single-byte-length-delimited
        // Deny response. Bodies are < 128 bytes so the varint is one byte.
        let server = std::thread::spawn(move || {
            let (mut stream, _) = listener.accept().expect("accept");

            let mut tag = [0u8; 1];
            stream.read_exact(&mut tag).expect("read heartbeat tag");
            assert_eq!(tag[0], codec::TAG_HEARTBEAT);

            stream.read_exact(&mut tag).expect("read query tag");
            assert_eq!(tag[0], codec::TAG_POLICY_QUERY);
            let mut len = [0u8; 1];
            stream.read_exact(&mut len).expect("read query len");
            if len[0] > 0 {
                let mut body = vec![0u8; len[0] as usize];
                stream.read_exact(&mut body).expect("read query body");
            }

            let resp = aa_proto::assembly::policy::v1::CheckActionResponse {
                decision: Decision::Deny as i32,
                reason: "blocked by test policy".to_owned(),
                ..Default::default()
            };
            let mut buf = Vec::new();
            resp.encode(&mut buf).expect("encode response");
            assert!(buf.len() < 128, "test assumes a single-byte length varint");
            stream
                .write_all(&[codec::TAG_POLICY_RESPONSE, buf.len() as u8])
                .expect("write resp header");
            stream.write_all(&buf).expect("write resp body");
            stream.flush().expect("flush");
            // Hold the connection open long enough for the client to read.
            std::thread::sleep(std::time::Duration::from_millis(300));
        });

        let endpoint = CString::new(socket_path.clone()).expect("valid endpoint");
        let mut client: *mut aa_client_handle = ptr::null_mut();
        // SAFETY: valid pointers from a controlled test context.
        assert_eq!(
            unsafe { aa_connect(endpoint.as_ptr(), &mut client) },
            AA_STATUS_OK
        );

        let agent = CString::new("agent-1").expect("valid");
        let action = CString::new("tool_call").expect("valid");
        let tool = CString::new("web_search").expect("valid");
        let mut decision: AaDecision = AA_DECISION_ALLOW;
        let mut reason: *mut c_char = ptr::null_mut();
        // SAFETY: handle from aa_connect; valid in/out pointers.
        let status = unsafe {
            aa_query_policy(
                client,
                agent.as_ptr(),
                action.as_ptr(),
                tool.as_ptr(),
                ptr::null(),
                &mut decision,
                &mut reason,
            )
        };

        assert_eq!(status, AA_STATUS_OK);
        assert_eq!(decision, AA_DECISION_DENY);
        assert!(!reason.is_null());
        // SAFETY: reason came from CString::into_raw in aa_query_policy.
        let reason_str = unsafe { CStr::from_ptr(reason) }
            .to_str()
            .expect("utf8")
            .to_owned();
        assert_eq!(reason_str, "blocked by test policy");
        // SAFETY: free the reason via the crate's own allocator helper.
        unsafe { aa_free_string(reason) };

        // SAFETY: handle from aa_connect, not yet disconnected.
        assert_eq!(unsafe { aa_disconnect(client) }, AA_STATUS_OK);

        server.join().expect("mock server thread");
        let _ = std::fs::remove_file(&socket_path);
    }

    /// With no runtime listening, the synchronous query times out; the binding
    /// must fail **open** — return OK with `AA_DECISION_ALLOW` — so an
    /// unreachable runtime never blocks the agent.
    #[test]
    fn query_policy_fails_open_with_no_server() {
        // No server binds this path, so the background IPC thread never
        // connects and the query times out (QueryFailed -> fail-open).
        let endpoint = CString::new(format!(
            "/tmp/aa-ffi-go-no-server-{}.sock",
            std::process::id()
        ))
        .expect("valid endpoint");
        let mut client: *mut aa_client_handle = ptr::null_mut();
        // SAFETY: valid pointers from a controlled test context.
        assert_eq!(
            unsafe { aa_connect(endpoint.as_ptr(), &mut client) },
            AA_STATUS_OK
        );

        let agent = CString::new("agent-1").expect("valid");
        let action = CString::new("tool_call").expect("valid");
        let mut decision: AaDecision = AA_DECISION_DENY;
        let mut reason: *mut c_char = ptr::null_mut();
        // SAFETY: handle from aa_connect; valid in/out pointers.
        let status = unsafe {
            aa_query_policy(
                client,
                agent.as_ptr(),
                action.as_ptr(),
                ptr::null(),
                ptr::null(),
                &mut decision,
                &mut reason,
            )
        };

        assert_eq!(
            status, AA_STATUS_OK,
            "fail-open returns OK, not an error status"
        );
        assert_eq!(
            decision, AA_DECISION_ALLOW,
            "unreachable runtime must fail open"
        );
        assert!(!reason.is_null());
        // SAFETY: reason came from CString::into_raw in aa_query_policy.
        unsafe { aa_free_string(reason) };

        // SAFETY: handle from aa_connect, not yet disconnected.
        assert_eq!(unsafe { aa_disconnect(client) }, AA_STATUS_OK);
    }
}

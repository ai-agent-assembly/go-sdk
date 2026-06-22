/*
 * GDPR cookie-consent opt-in banner (AAASM-3554).
 *
 * Dependency-free vanilla JS. Pairs with the Consent Mode v2 default-denied
 * init in layouts/_partials/custom/head-end.html. The banner lets a visitor
 * opt in to (or reject) analytics:
 *   Accept -> gtag('consent','update',{analytics_storage:'granted'})
 *             + persist 'granted' + hide.
 *   Reject -> persist 'denied' + hide.
 * If a choice is already stored, the banner is never shown.
 *
 * Built with document.createElement + textContent (never innerHTML) and
 * appended to document.body. Non-modal, fixed at the bottom, keyboard
 * accessible (role=dialog, focusable buttons, Esc rejects).
 */
(function () {
  "use strict";

  var STORAGE_KEY = "aa-analytics-consent";

  function readChoice() {
    try {
      return window.localStorage ? localStorage.getItem(STORAGE_KEY) : null;
    } catch (e) {
      return null;
    }
  }

  function persist(value) {
    try {
      if (window.localStorage) {
        localStorage.setItem(STORAGE_KEY, value);
      }
    } catch (e) {
      /* localStorage unavailable (private mode); choice not persisted */
    }
  }

  function gtagSafe() {
    if (typeof window.gtag === "function") {
      window.gtag.apply(window, arguments);
    }
  }

  function build() {
    // Don't show the banner if the visitor already chose.
    if (readChoice() !== null) {
      return;
    }

    var banner = document.createElement("div");
    banner.className = "aa-consent-banner";
    banner.setAttribute("role", "dialog");
    banner.setAttribute("aria-modal", "false");
    banner.setAttribute("aria-live", "polite");
    banner.setAttribute("aria-label", "Analytics cookie consent");

    var text = document.createElement("p");
    text.className = "aa-consent-text";
    text.textContent =
      "We use Google Analytics to understand how the docs are used. " +
      "Analytics cookies are off until you accept.";

    var actions = document.createElement("div");
    actions.className = "aa-consent-actions";

    var accept = document.createElement("button");
    accept.type = "button";
    accept.className = "aa-consent-btn aa-consent-accept";
    accept.textContent = "Accept";

    var reject = document.createElement("button");
    reject.type = "button";
    reject.className = "aa-consent-btn aa-consent-reject";
    reject.textContent = "Reject";

    function close() {
      if (banner.parentNode) {
        banner.parentNode.removeChild(banner);
      }
    }

    accept.addEventListener("click", function () {
      gtagSafe("consent", "update", { analytics_storage: "granted" });
      persist("granted");
      close();
    });

    reject.addEventListener("click", function () {
      persist("denied");
      close();
    });

    banner.addEventListener("keydown", function (event) {
      if (event.key === "Escape" || event.key === "Esc") {
        persist("denied");
        close();
      }
    });

    actions.appendChild(accept);
    actions.appendChild(reject);
    banner.appendChild(text);
    banner.appendChild(actions);
    document.body.appendChild(banner);

    // Move keyboard focus to the primary action for accessibility.
    accept.focus();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", build);
  } else {
    build();
  }
})();

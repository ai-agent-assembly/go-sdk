/*
 * "Was this page helpful?" page-feedback widget (AAASM-3555).
 *
 * Dependency-free vanilla JS. Builds two keyboard-accessible buttons
 * (👍 / 👎) under a heading and appends them to the container rendered by
 * layouts/_partials/custom/feedback-widget.html (or, as a fallback, to the
 * page's <main> element). On click it fires a GA4 'feedback' event via
 * window.gtag — guarded by a typeof check so it is a no-op when GA is absent
 * (dev/preview builds, or before consent loads gtag). Consent Mode v2 (set up
 * in custom/head-end.html, AAASM-3554) gates analytics storage automatically,
 * so the event is fired regardless of the stored consent choice.
 *
 *   👍 -> gtag('event','feedback',{ value: 1, page_path: location.pathname })
 *   👎 -> gtag('event','feedback',{ value: 0, page_path: location.pathname })
 *         + reveal a "Tell us what we can improve →" link that opens a
 *           pre-filled GitHub issue in a new tab (rel=noopener).
 *
 * After either click the buttons are replaced with "Thanks for your feedback!".
 *
 * Built with document.createElement + textContent (never innerHTML of dynamic
 * data) so it stays CodeQL-clean.
 */
(function () {
  "use strict";

  var ISSUE_BASE =
    "https://github.com/ai-agent-assembly/go-sdk/issues/new";

  function gtagSafe() {
    if (typeof window.gtag === "function") {
      window.gtag.apply(window, arguments);
    }
  }

  function improveURL() {
    var title = "Docs feedback: " + window.location.pathname;
    var body =
      "Page: " + window.location.href + "\n\nWhat could be improved?\n";
    return (
      ISSUE_BASE +
      "?labels=" +
      encodeURIComponent("docs,feedback") +
      "&title=" +
      encodeURIComponent(title) +
      "&body=" +
      encodeURIComponent(body)
    );
  }

  function sendFeedback(value) {
    gtagSafe("event", "feedback", {
      value: value,
      page_path: window.location.pathname,
    });
  }

  function build() {
    var container = document.getElementById("aa-feedback");
    if (!container) {
      // Fallback: no partial container on the page, attach to <main>.
      var main = document.getElementById("content") ||
        document.querySelector("main");
      if (!main) {
        return;
      }
      container = document.createElement("div");
      container.id = "aa-feedback";
      container.className = "aa-feedback";
      main.appendChild(container);
    }

    // Guard against double-init (e.g. partial container + this script rerun).
    if (container.getAttribute("data-aa-feedback-ready") === "1") {
      return;
    }
    container.setAttribute("data-aa-feedback-ready", "1");

    var heading = document.createElement("p");
    heading.className = "aa-feedback-heading";
    heading.textContent = "Was this page helpful?";

    var actions = document.createElement("div");
    actions.className = "aa-feedback-actions";

    var up = document.createElement("button");
    up.type = "button";
    up.className = "aa-feedback-btn aa-feedback-up";
    up.textContent = "👍";
    up.setAttribute("aria-label", "Yes, this page was helpful");

    var down = document.createElement("button");
    down.type = "button";
    down.className = "aa-feedback-btn aa-feedback-down";
    down.textContent = "👎";
    down.setAttribute("aria-label", "No, this page was not helpful");

    function thanks(showImprove) {
      // Replace the buttons with a thank-you message.
      while (container.firstChild) {
        container.removeChild(container.firstChild);
      }

      var msg = document.createElement("p");
      msg.className = "aa-feedback-thanks";
      msg.textContent = "Thanks for your feedback!";
      container.appendChild(msg);

      if (showImprove) {
        var link = document.createElement("a");
        link.className = "aa-feedback-improve";
        link.href = improveURL();
        link.target = "_blank";
        link.rel = "noopener";
        link.textContent = "Tell us what we can improve →";
        container.appendChild(link);
      }
    }

    up.addEventListener("click", function () {
      sendFeedback(1);
      thanks(false);
    });

    down.addEventListener("click", function () {
      sendFeedback(0);
      thanks(true);
    });

    actions.appendChild(up);
    actions.appendChild(down);
    container.appendChild(heading);
    container.appendChild(actions);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", build);
  } else {
    build();
  }
})();

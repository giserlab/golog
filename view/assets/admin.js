/**
 * Admin Native JS
 * Replaces tocass.js for the admin panel.
 * Covers only dropdown and tooltip interactions.
 */

(function () {
  "use strict";

  /* ===============================
     Dropdown
     =============================== */

  function initDropdowns() {
    document.addEventListener("click", function (e) {
      const trigger = e.target.closest("[data-dropdown]");
      if (trigger) {
        e.preventDefault();
        const id = trigger.getAttribute("data-dropdown");
        const dropdown = document.getElementById(id);
        if (!dropdown) return;

        const isOpen = dropdown.classList.contains("is-open");
        closeAllDropdowns();
        if (!isOpen) {
          dropdown.classList.add("is-open");
          positionDropdown(dropdown, trigger);
        }
        return;
      }

      if (!e.target.closest(".ts-dropdown")) {
        closeAllDropdowns();
      }
    });
  }

  function closeAllDropdowns() {
    document.querySelectorAll(".ts-dropdown.is-open").forEach(function (el) {
      el.classList.remove("is-open");
    });
  }

  function positionDropdown(dropdown, trigger) {
    const rect = trigger.getBoundingClientRect();
    const ddHeight = dropdown.offsetHeight;
    const ddWidth = dropdown.offsetWidth;
    const viewportW = window.innerWidth;
    const viewportH = window.innerHeight;

    let top = rect.bottom + 4;
    let left = rect.left;

    if (left + ddWidth > viewportW) {
      left = rect.right - ddWidth;
    }
    if (top + ddHeight > viewportH) {
      top = rect.top - ddHeight - 4;
    }

    dropdown.style.top = top + window.scrollY + "px";
    dropdown.style.left = left + window.scrollX + "px";
  }

  /* ===============================
     Tooltip
     =============================== */

  let tooltipEl = null;

  function initTooltips() {
    document.addEventListener("mouseover", function (e) {
      const target = e.target.closest("[data-tooltip]");
      if (!target) return;
      showTooltip(target);
    });

    document.addEventListener("mouseout", function (e) {
      const target = e.target.closest("[data-tooltip]");
      if (!target) return;
      hideTooltip();
    });

    document.addEventListener("focusin", function (e) {
      const target = e.target.closest("[data-tooltip]");
      if (!target) return;
      showTooltip(target);
    });

    document.addEventListener("focusout", function (e) {
      const target = e.target.closest("[data-tooltip]");
      if (!target) return;
      hideTooltip();
    });
  }

  function showTooltip(target) {
    hideTooltip();
    const text = target.getAttribute("data-tooltip");
    if (!text) return;

    tooltipEl = document.createElement("div");
    tooltipEl.className = "admin-tooltip";
    tooltipEl.textContent = text;
    tooltipEl.style.cssText =
      "position:fixed;z-index:104;padding:0.35rem 0.6rem;" +
      "background:rgba(0,0,0,0.85);color:#fff;font-size:12px;" +
      "border-radius:0.3rem;white-space:nowrap;pointer-events:none;" +
      "opacity:0;transition:opacity 0.15s;";
    document.body.appendChild(tooltipEl);

    const rect = target.getBoundingClientRect();
    const ttRect = tooltipEl.getBoundingClientRect();

    let top = rect.top - ttRect.height - 6;
    let left = rect.left + rect.width / 2 - ttRect.width / 2;

    if (left < 4) left = 4;
    if (left + ttRect.width > window.innerWidth - 4) {
      left = window.innerWidth - ttRect.width - 4;
    }
    if (top < 4) {
      top = rect.bottom + 6;
    }

    tooltipEl.style.top = top + "px";
    tooltipEl.style.left = left + "px";

    requestAnimationFrame(function () {
      if (tooltipEl) tooltipEl.style.opacity = "1";
    });
  }

  function hideTooltip() {
    if (tooltipEl) {
      tooltipEl.remove();
      tooltipEl = null;
    }
  }

  /* ===============================
     Mobile Sidebar
     =============================== */

  function initSidebar() {
    var toggle = document.getElementById("sidebar-toggle");
    var sidebar = document.getElementById("admin-sidebar");
    var close = document.getElementById("sidebar-close");
    var backdrop = document.getElementById("sidebar-backdrop");

    if (!toggle || !sidebar) return;

    function openSidebar() {
      sidebar.classList.add("is-open");
      if (backdrop) backdrop.classList.add("is-open");
      document.body.style.overflow = "hidden";
    }

    function closeSidebar() {
      sidebar.classList.remove("is-open");
      if (backdrop) backdrop.classList.remove("is-open");
      document.body.style.overflow = "";
    }

    toggle.addEventListener("click", function (e) {
      e.stopPropagation();
      if (sidebar.classList.contains("is-open")) {
        closeSidebar();
      } else {
        openSidebar();
      }
    });

    if (close) {
      close.addEventListener("click", closeSidebar);
    }

    if (backdrop) {
      backdrop.addEventListener("click", closeSidebar);
    }

    // Close on Escape key
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && sidebar.classList.contains("is-open")) {
        closeSidebar();
      }
    });

    // Close sidebar when clicking a nav link (on mobile)
    sidebar.querySelectorAll(".item").forEach(function (link) {
      link.addEventListener("click", function () {
        if (window.innerWidth <= 768) {
          closeSidebar();
        }
      });
    });
  }

  /* ===============================
     Mobile Filter Sidebar
     =============================== */

  function initMobileFilters() {
    // Close filter <details> on mobile by default
    if (window.innerWidth <= 768) {
      document.querySelectorAll(".admin-filter-details").forEach(function (el) {
        el.removeAttribute("open");
      });
    }
  }

  /* ===============================
     Init
     =============================== */

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () {
      initDropdowns();
      initTooltips();
      initSidebar();
      initMobileFilters();
    });
  } else {
    initDropdowns();
    initTooltips();
    initSidebar();
    initMobileFilters();
  }
})();

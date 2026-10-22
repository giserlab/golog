(function () {
  const MIN_SCALE = 0.25;
  const MAX_SCALE = 5;
  const ZOOM_STEPS = [0.25, 0.5, 0.75, 1, 1.25, 1.5, 2, 2.5, 3, 4, 5];

  // Current lightbox teardown, so opening another image can dispose the old one.
  let currentCleanup = null;

  function clampScale(value) {
    return Math.min(MAX_SCALE, Math.max(MIN_SCALE, value));
  }

  function nextStep(scale, direction) {
    if (direction > 0) {
      for (let i = 0; i < ZOOM_STEPS.length; i++) {
        if (ZOOM_STEPS[i] > scale + 0.001) {
          return ZOOM_STEPS[i];
        }
      }
      return MAX_SCALE;
    }
    if (direction < 0) {
      for (let i = ZOOM_STEPS.length - 1; i >= 0; i--) {
        if (ZOOM_STEPS[i] < scale - 0.001) {
          return ZOOM_STEPS[i];
        }
      }
    }
    return MIN_SCALE;
  }

  function createButton(action, label, title) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "golog-lightbox-btn";
    button.dataset.action = action;
    button.textContent = label;
    button.title = title;
    button.setAttribute("aria-label", title);
    return button;
  }

  function openLightbox(src) {
    if (currentCleanup) {
      currentCleanup();
      currentCleanup = null;
    }

    const overlay = document.createElement("div");
    overlay.className = "golog-lightbox";
    overlay.setAttribute("role", "dialog");
    overlay.setAttribute("aria-modal", "true");
    overlay.setAttribute("aria-label", "图片预览");

    const img = document.createElement("img");
    img.className = "golog-lightbox-img";
    img.src = src;
    img.alt = "";
    img.draggable = false;

    const toolbar = document.createElement("div");
    toolbar.className = "golog-lightbox-toolbar";
    toolbar.setAttribute("role", "toolbar");
    toolbar.setAttribute("aria-label", "图片缩放");

    const zoomOutBtn = createButton("zoom-out", "−", "缩小 (-)");
    const scaleLabel = document.createElement("span");
    scaleLabel.className = "golog-lightbox-scale";
    scaleLabel.textContent = "100%";
    scaleLabel.setAttribute("aria-live", "polite");
    const zoomInBtn = createButton("zoom-in", "+", "放大 (+)");
    const resetBtn = createButton("reset", "重置", "重置大小 (0)");

    toolbar.appendChild(zoomOutBtn);
    toolbar.appendChild(scaleLabel);
    toolbar.appendChild(zoomInBtn);
    toolbar.appendChild(resetBtn);

    const close = document.createElement("button");
    close.type = "button";
    close.className = "golog-lightbox-close";
    close.innerHTML = "&times;";
    close.setAttribute("aria-label", "关闭");

    overlay.appendChild(img);
    overlay.appendChild(toolbar);
    overlay.appendChild(close);
    document.body.appendChild(overlay);

    // Lock page scrolling while the preview is open and restore it on close.
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    let scale = 1;
    let tx = 0;
    let ty = 0;
    let dragging = false;
    let dragOrigin = null;
    let pinchOrigin = null;
    let closing = false;
    let detached = false;
    const pointers = new Map();

    function updateScaleLabel() {
      scaleLabel.textContent = Math.round(scale * 100) + "%";
      resetBtn.disabled = scale === 1 && tx === 0 && ty === 0;
      overlay.classList.toggle("is-zoomed", scale > 1 + 0.001);
    }

    // Keep the image from being dragged/panned off the viewport.
    function clampPan() {
      if (scale <= 1) {
        tx = 0;
        ty = 0;
        return;
      }
      const maxX = Math.max(0, (img.clientWidth * scale - overlay.clientWidth) / 2);
      const maxY = Math.max(0, (img.clientHeight * scale - overlay.clientHeight) / 2);
      tx = Math.min(maxX, Math.max(-maxX, tx));
      ty = Math.min(maxY, Math.max(-maxY, ty));
    }

    function applyTransform() {
      img.style.transform = "translate(" + tx + "px, " + ty + "px) scale(" + scale + ")";
      updateScaleLabel();
    }

    // Convert viewport coordinates to coordinates relative to the image center.
    function centerPoint(clientX, clientY) {
      const rect = overlay.getBoundingClientRect();
      return {
        x: clientX - rect.left - rect.width / 2,
        y: clientY - rect.top - rect.height / 2,
      };
    }

    function zoomTo(targetScale, point, animate) {
      const next = clampScale(targetScale);
      if (Math.abs(next - scale) < 0.001) {
        return;
      }
      // Keep the point under the cursor/fingers anchored while scaling.
      const ratio = next / scale;
      tx = point.x - (point.x - tx) * ratio;
      ty = point.y - (point.y - ty) * ratio;
      scale = next;
      clampPan();
      img.classList.toggle("is-animating", Boolean(animate));
      applyTransform();
    }

    function resetView(animate) {
      scale = 1;
      tx = 0;
      ty = 0;
      img.classList.toggle("is-animating", Boolean(animate));
      applyTransform();
    }

    function zoomStep(direction) {
      zoomTo(nextStep(scale, direction), { x: 0, y: 0 }, true);
    }

    function pointerList() {
      return Array.from(pointers.values());
    }

    function pinchInfo() {
      const [a, b] = pointerList();
      const rect = overlay.getBoundingClientRect();
      return {
        mid: {
          x: (a.x + b.x) / 2 - rect.left - rect.width / 2,
          y: (a.y + b.y) / 2 - rect.top - rect.height / 2,
        },
        distance: Math.hypot(a.x - b.x, a.y - b.y) || 1,
      };
    }

    function startPinch() {
      const info = pinchInfo();
      pinchOrigin = { scale: scale, tx: tx, ty: ty, mid: info.mid, distance: info.distance };
    }

    function updatePinch() {
      if (!pinchOrigin) {
        return;
      }
      const info = pinchInfo();
      const next = clampScale(pinchOrigin.scale * (info.distance / pinchOrigin.distance));
      const ratio = next / pinchOrigin.scale;
      scale = next;
      tx = info.mid.x - (pinchOrigin.mid.x - pinchOrigin.tx) * ratio;
      ty = info.mid.y - (pinchOrigin.mid.y - pinchOrigin.ty) * ratio;
      clampPan();
      applyTransform();
    }

    function onPointerDown(e) {
      if (e.pointerType === "mouse" && e.button !== 0) {
        return;
      }
      pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
      img.classList.remove("is-animating");
      if (pointers.size === 1) {
        dragging = scale > 1.001;
        dragOrigin = { x: e.clientX - tx, y: e.clientY - ty };
        img.classList.toggle("is-dragging", dragging);
      } else if (pointers.size === 2) {
        dragging = false;
        dragOrigin = null;
        img.classList.remove("is-dragging");
        startPinch();
      }
      try {
        img.setPointerCapture(e.pointerId);
      } catch (err) {
        // Pointer capture is best-effort; dragging still works without it.
      }
    }

    function onPointerMove(e) {
      const pointer = pointers.get(e.pointerId);
      if (!pointer) {
        return;
      }
      pointer.x = e.clientX;
      pointer.y = e.clientY;
      if (pointers.size === 1 && dragging) {
        tx = e.clientX - dragOrigin.x;
        ty = e.clientY - dragOrigin.y;
        clampPan();
        applyTransform();
      } else if (pointers.size === 2) {
        updatePinch();
      }
    }

    function onPointerUp(e) {
      if (!pointers.has(e.pointerId)) {
        return;
      }
      pointers.delete(e.pointerId);
      if (pointers.size === 1) {
        // Falling back from pinch to a single finger: continue as a drag.
        const remaining = pointerList()[0];
        dragOrigin = { x: remaining.x - tx, y: remaining.y - ty };
        dragging = scale > 1.001;
        pinchOrigin = null;
        img.classList.toggle("is-dragging", dragging);
      } else if (pointers.size === 0) {
        dragging = false;
        dragOrigin = null;
        pinchOrigin = null;
        img.classList.remove("is-dragging");
      }
    }

    function onWheel(e) {
      e.preventDefault();
      if (!e.deltaY) {
        return;
      }
      const delta = e.deltaMode === 1 ? e.deltaY * 16 : e.deltaY;
      const point = centerPoint(e.clientX, e.clientY);
      zoomTo(scale * Math.exp(-delta * 0.0015), point, false);
    }

    function onDoubleClick(e) {
      e.preventDefault();
      if (scale > 1.001) {
        resetView(true);
      } else {
        zoomTo(2.5, centerPoint(e.clientX, e.clientY), true);
      }
    }

    function onToolbarClick(e) {
      const button = e.target.closest("button[data-action]");
      if (!button || !toolbar.contains(button)) {
        return;
      }
      if (button.dataset.action === "zoom-in") {
        zoomStep(1);
      } else if (button.dataset.action === "zoom-out") {
        zoomStep(-1);
      } else if (button.dataset.action === "reset") {
        resetView(true);
      }
    }

    function onOverlayClick(e) {
      if (e.target === overlay || e.target === close) {
        closeLightbox();
      }
    }

    function onKeyDown(e) {
      if (e.key === "Escape") {
        e.preventDefault();
        closeLightbox();
      } else if (e.key === "+" || e.key === "=") {
        e.preventDefault();
        zoomStep(1);
      } else if (e.key === "-" || e.key === "_") {
        e.preventDefault();
        zoomStep(-1);
      } else if (e.key === "0") {
        e.preventDefault();
        resetView(true);
      }
    }

    function onResize() {
      clampPan();
      applyTransform();
    }

    function detach() {
      if (detached) {
        return;
      }
      detached = true;
      document.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("resize", onResize);
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", onPointerUp);
      window.removeEventListener("pointercancel", onPointerUp);
      document.body.style.overflow = previousOverflow;
    }

    function destroy() {
      detach();
      overlay.remove();
    }

    function closeLightbox() {
      if (closing) {
        return;
      }
      closing = true;
      overlay.classList.remove("active");
      detach();
      setTimeout(destroy, 250);
      if (currentCleanup === destroy) {
        currentCleanup = null;
      }
    }

    currentCleanup = destroy;

    img.addEventListener("pointerdown", onPointerDown);
    img.addEventListener("dblclick", onDoubleClick);
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", onPointerUp);
    window.addEventListener("pointercancel", onPointerUp);
    window.addEventListener("resize", onResize);
    overlay.addEventListener("wheel", onWheel, { passive: false });
    overlay.addEventListener("click", onOverlayClick);
    toolbar.addEventListener("click", onToolbarClick);
    document.addEventListener("keydown", onKeyDown);

    // Trigger reflow so the transition plays.
    void overlay.offsetWidth;
    overlay.classList.add("active");
    applyTransform();
  }

  function initLightbox(selector) {
    const images = document.querySelectorAll(selector);
    images.forEach(function (img) {
      // Skip images already wrapped in a link so the original link behavior is preserved.
      if (img.closest("a")) {
        return;
      }
      img.classList.add("golog-lightbox-trigger");
      img.addEventListener("click", function (e) {
        e.preventDefault();
        const src = img.getAttribute("data-src") || img.src;
        if (src) {
          openLightbox(src);
        }
      });
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    // Blog post content
    initLightbox(".markdown-body img");
    // Whisper / log content
    initLightbox(".intro img");
    // Moment covers
    initLightbox(".cover img");
  });
})();

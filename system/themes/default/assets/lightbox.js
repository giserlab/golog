(function () {
  function openLightbox(src) {
    const existing = document.querySelector('.golog-lightbox');
    if (existing) {
      existing.remove();
    }

    const overlay = document.createElement('div');
    overlay.className = 'golog-lightbox';
    overlay.setAttribute('role', 'dialog');
    overlay.setAttribute('aria-modal', 'true');

    const img = document.createElement('img');
    img.src = src;
    img.alt = '';

    const close = document.createElement('span');
    close.className = 'golog-lightbox-close';
    close.innerHTML = '&times;';
    close.setAttribute('aria-label', '关闭');

    overlay.appendChild(img);
    overlay.appendChild(close);
    document.body.appendChild(overlay);

    // Trigger reflow so the transition plays.
    void overlay.offsetWidth;
    overlay.classList.add('active');

    function closeLightbox() {
      overlay.classList.remove('active');
      setTimeout(() => overlay.remove(), 250);
    }

    overlay.addEventListener('click', function (e) {
      if (e.target === overlay || e.target === close) {
        closeLightbox();
      }
    });

    document.addEventListener('keydown', function onKey(e) {
      if (e.key === 'Escape') {
        closeLightbox();
        document.removeEventListener('keydown', onKey);
      }
    });
  }

  function initLightbox(selector) {
    const images = document.querySelectorAll(selector);
    images.forEach(function (img) {
      // Skip images already wrapped in a link so the original link behavior is preserved.
      if (img.closest('a')) {
        return;
      }
      img.classList.add('golog-lightbox-trigger');
      img.addEventListener('click', function (e) {
        e.preventDefault();
        const src = img.getAttribute('data-src') || img.src;
        if (src) {
          openLightbox(src);
        }
      });
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    // Blog post content
    initLightbox('.markdown-body img');
    // Whisper / log content
    initLightbox('.intro img');
    // Moment covers
    initLightbox('.cover img');
  });
})();

/* 企业博客主题 · 交互脚本
   图片懒加载渐显：进入视口前 240px 预加载，加载完成后淡入。
   缺少对应 DOM 时直接跳过。 */
(function () {
  "use strict";

  /* 兜底占位图：当图片地址缺失或加载失败时使用，避免出现裂图。 */
  var FALLBACK =
    "data:image/svg+xml;charset=utf-8," +
    encodeURIComponent(
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#b6bdc9" stroke-width="1.6" stroke-linecap="round">' +
        '<rect x="3.5" y="5" width="17" height="14" rx="2.5"/>' +
        '<path d="M3.5 15.5l4.6-4.2 4 3.6 3.2-3 5.2 4.6"/>' +
        "</svg>"
    );

  /* ---------- 图片懒加载渐显 ---------- */
  function initLazyImages() {
    // 只接管带 lazy-img 标记的主题图片，其余图片（头像、favicon 等）保持原样。
    // 注意：选择器不能带 [data-src]——若有其它脚本抢先移除了该属性，
    // 图片就会既没被接管、又因动画停在 opacity:0，从而永久不可见。
    var images = Array.prototype.slice.call(document.querySelectorAll("img.lazy-img"));
    if (!images.length) return;

    // 取出真实地址并立刻清除 data-src：
    // 这样即使页面上还有其它脚本（或浏览器扩展）观察同一批图片，
    // 也不会再次读到该属性，更不会把 src 覆盖成字符串 "null"。
    function takeSource(img) {
      var src = img.getAttribute("data-src");
      img.removeAttribute("data-src");
      return src && src !== "null" && src !== "undefined" ? src : "";
    }

    function finish(img, ok) {
      var src = img.getAttribute("src");
      if (!ok && (!src || src === "null")) {
        img.setAttribute("src", FALLBACK);
      }
      img.classList.add("is-loaded");
    }

    function load(img) {
      var src = takeSource(img);
      if (!src) {
        finish(img, false);
        return;
      }
      img.addEventListener("load", function () { finish(img, true); }, { once: true });
      img.addEventListener("error", function () { finish(img, false); }, { once: true });
      img.src = src;
    }

    // 未进入视口的图片保持骨架占位，滚动到附近再加载。
    if (!("IntersectionObserver" in window)) {
      images.forEach(load);
      return;
    }

    var observer = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (!entry.isIntersecting) return;
          observer.unobserve(entry.target);
          load(entry.target);
        });
      },
      { rootMargin: "240px 0px", threshold: 0.01 }
    );

    images.forEach(function (img) {
      observer.observe(img);
    });
  }

  function boot() {
    initLazyImages();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();

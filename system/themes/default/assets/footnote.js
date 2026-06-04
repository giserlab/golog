(function () {
  function showFootnoteTooltip(link) {
    const index = link.dataset.index;
    if (!index) return;

    const fnLi = document.getElementById("fn:" + index);
    if (!fnLi) return;

    let content = "";
    const children = fnLi.children;
    for (let i = 0; i < children.length; i++) {
      const clone = children[i].cloneNode(true);
      clone.querySelectorAll(".footnote-backref").forEach(function (b) {
        b.remove();
      });
      content += clone.outerHTML;
    }
    if (!content) {
      const tmp = document.createElement("div");
      tmp.innerHTML = fnLi.innerHTML;
      tmp.querySelectorAll(".footnote-backref").forEach(function (b) {
        b.remove();
      });
      content = tmp.innerHTML.trim();
    }

    let tooltip = document.getElementById("footnote-tooltip");
    if (!tooltip) {
      tooltip = document.createElement("div");
      tooltip.id = "footnote-tooltip";
      tooltip.className = "footnote-tooltip";
      document.body.appendChild(tooltip);
    }

    tooltip.innerHTML = content;

    const rect = link.getBoundingClientRect();
    tooltip.classList.add("visible");

    const tooltipRect = tooltip.getBoundingClientRect();
    let left = rect.left + rect.width / 2 - tooltipRect.width / 2;
    let top = rect.top - tooltipRect.height - 10;

    if (left < 8) left = 8;
    if (left + tooltipRect.width > window.innerWidth - 8) {
      left = window.innerWidth - tooltipRect.width - 8;
    }
    if (top < 8) {
      top = rect.bottom + 10;
    }

    tooltip.style.left = left + window.scrollX + "px";
    tooltip.style.top = top + window.scrollY + "px";

    function closeOnClickOutside(e) {
      if (!tooltip.contains(e.target) && e.target !== link) {
        tooltip.classList.remove("visible");
        document.removeEventListener("click", closeOnClickOutside);
      }
    }
    setTimeout(function () {
      document.addEventListener("click", closeOnClickOutside);
    }, 0);
  }

  document.addEventListener("click", function (e) {
    const link = e.target.closest('.footnote-ref[role="doc-noteref"]');
    if (link) {
      e.preventDefault();
      showFootnoteTooltip(link);
    }
  });
})();

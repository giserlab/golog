/**
 * 留言回复交互（所有主题共用，由 AssetView 以 /assets/comment.js 提供）。
 *
 * 页面上只有一个留言表单。点击某条留言旁的「回复」按钮时：
 *   1. 把留言 ID 写入表单的 parent_id 隐藏字段；
 *   2. 在表单顶部显示「正在回复 XXX」提示条（可取消）；
 *   3. 设置正文输入框的占位提示（例如「回复 @XXX：」）；
 *   4. 滚动到表单并聚焦正文输入框。
 *
 * 这里刻意不给被回复的留言加高亮：留言列表本身不该因为“正在回复谁”而变化，
 * 提示信息统一集中在表单一侧，避免用户把高亮误读成“已选中/已读”。
 *
 * 之所以复用同一个表单，是因为每个表单都要内嵌一个 ALTCHA widget，
 * 若每条留言各有一个表单，访客每回复一次就要重新解一次 PoW 挑战。
 */
(function () {
  function ready(fn) {
    if (document.readyState !== 'loading') {
      fn();
    } else {
      document.addEventListener('DOMContentLoaded', fn);
    }
  }

  ready(function () {
    var form = document.querySelector('.comment-form');
    if (!form) return;

    var parentInput = form.querySelector('input[name="parent_id"]');
    if (!parentInput) return;

    var hint = form.querySelector('.comment-reply-hint');
    var nameEl = hint ? hint.querySelector('.comment-reply-name') : null;
    var cancel = hint ? hint.querySelector('.comment-reply-cancel') : null;
    var content = form.querySelector('textarea[name="content"]');
    var links = document.querySelectorAll('.comment-reply-link');
    // 提示条上的本地化前缀模板，例如「回复 @%s：」
    var prefix = hint ? hint.getAttribute('data-reply-placeholder') || '' : '';
    var defaultPlaceholder = content ? content.placeholder : '';

    function clearReply() {
      parentInput.value = '';
      if (nameEl) nameEl.textContent = '';
      if (hint) hint.hidden = true;
      if (content) content.placeholder = defaultPlaceholder;
    }

    Array.prototype.forEach.call(links, function (link) {
      link.addEventListener('click', function (event) {
        event.preventDefault();
        var author = link.getAttribute('data-comment-author') || '';

        parentInput.value = link.getAttribute('data-comment-id') || '';
        if (nameEl) nameEl.textContent = author;
        if (hint) hint.hidden = false;
        if (content && prefix) content.placeholder = prefix.replace('%s', author);

        form.scrollIntoView({ behavior: 'smooth', block: 'center' });
        if (content) content.focus({ preventScroll: true });
      });
    });

    if (cancel) {
      cancel.addEventListener('click', function (event) {
        event.preventDefault();
        clearReply();
      });
    }
  });
})();

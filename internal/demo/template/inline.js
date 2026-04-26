// SharkAuth demo report — collapse + copy helpers (no deps)
(function () {
  document.querySelectorAll('.jwt-toggle').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var detail = btn.closest('.card').querySelector('.jwt-detail');
      if (!detail) return;
      var open = detail.classList.toggle('open');
      btn.textContent = open ? 'hide JWT' : 'show JWT';
    });
  });

  document.querySelectorAll('.copy-btn').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var target = btn.closest('.jwt-detail');
      if (!target) return;
      var text = target.querySelector('.jwt-text');
      var content = text ? text.textContent : target.textContent;
      navigator.clipboard.writeText(content.trim()).then(function () {
        var orig = btn.textContent;
        btn.textContent = 'copied!';
        setTimeout(function () { btn.textContent = orig; }, 1200);
      });
    });
  });
}());

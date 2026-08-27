// Hash-based router: #/ -> dashboard, #/pr/:owner/:repo/:number -> detail view.
(function () {
  function route() {
    const hash = location.hash.replace(/^#/, "") || "/";
    const detailMatch = hash.match(/^\/pr\/([^/]+)\/([^/]+)\/(\d+)$/);
    if (detailMatch) {
      const [, owner, repo, number] = detailMatch;
      Detail.load(decodeURIComponent(owner), decodeURIComponent(repo), Number(number));
      return;
    }
    Dashboard.reload();
  }

  window.addEventListener("hashchange", route);
  window.addEventListener("DOMContentLoaded", route);
})();

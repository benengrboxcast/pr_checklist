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

  async function loadViewerBadge() {
    const badge = document.getElementById("viewer-badge");
    try {
      const { login } = await Api.getViewer();
      badge.textContent = `Signed in as ${login}`;
    } catch (e) {
      badge.textContent = "Not signed in to GitHub";
      badge.title = e.message;
    }
  }

  window.addEventListener("hashchange", route);
  window.addEventListener("DOMContentLoaded", async () => {
    loadViewerBadge(); // independent of tag titles; don't block the first render on it
    await TagTitles.load();
    route();
  });
})();

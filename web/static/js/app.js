// Hash-based router: #/ -> dashboard ("Opened by me"), #/pr/:owner/:repo/:number
// -> detail view, #/review-requests -> the "Review requested" view.
(function () {
  function route() {
    const hash = location.hash.replace(/^#/, "") || "/";
    updateViewTabs(hash);

    const detailMatch = hash.match(/^\/pr\/([^/]+)\/([^/]+)\/(\d+)$/);
    if (detailMatch) {
      const [, owner, repo, number] = detailMatch;
      Detail.load(decodeURIComponent(owner), decodeURIComponent(repo), Number(number));
      return;
    }
    if (hash === "/review-requests") {
      ReviewRequests.load();
      return;
    }
    Dashboard.reload();
  }

  // Highlights whichever top-level tab matches the current view. A PR detail
  // page counts as part of "Opened by me" since it's a drill-down of it.
  function updateViewTabs(hash) {
    const dashboardTab = document.getElementById("tab-dashboard");
    const reviewsTab = document.getElementById("tab-reviews");
    if (!dashboardTab || !reviewsTab) return;
    const onReviews = hash === "/review-requests";
    dashboardTab.classList.toggle("active", !onReviews);
    reviewsTab.classList.toggle("active", onReviews);
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

  document.getElementById("tab-dashboard").addEventListener("click", () => {
    location.hash = "/";
  });
  document.getElementById("tab-reviews").addEventListener("click", () => {
    location.hash = "/review-requests";
  });

  window.addEventListener("hashchange", route);
  window.addEventListener("DOMContentLoaded", async () => {
    loadViewerBadge(); // independent of tag titles/repo groups; don't block the first render on it
    await Promise.all([TagTitles.load(), RepoGroups.load()]);
    route();
  });
})();

// "Review requested" view: a live, unstored list of PRs where you've been
// asked to review (or have already reviewed). Fetched fresh from GitHub
// every time this view loads — nothing here is tracked/persisted locally,
// unlike the "Opened by me" dashboard.
const ReviewRequests = (() => {
  function statusBadge(status) {
    if (status === "review_requested") return `<span class="badge amber">Review Requested</span>`;
    return `<span class="badge green">Reviewed</span>`;
  }

  function row(pr) {
    return `
      <div class="pr-card">
        <div class="pr-main">
          <a class="pr-title" href="${pr.url}" target="_blank" rel="noopener">${Render.escapeHtml(pr.title)}</a>
          <div class="pr-meta">${Render.escapeHtml(pr.owner)}/${Render.escapeHtml(pr.repo)}#${pr.number}</div>
        </div>
        ${statusBadge(pr.status)}
      </div>`;
  }

  function statsBarHtml(prs) {
    const total = prs.length;
    const needsReview = prs.filter((pr) => pr.status === "review_requested").length;
    return `
      <div class="stats-bar">
        <span class="stat-total">${total} PR${total === 1 ? "" : "s"}</span>
        <span class="badge amber">${needsReview} Need Your Review</span>
      </div>`;
  }

  function render(prs) {
    const html = `
      ${statsBarHtml(prs)}
      <div class="toolbar">
        <button id="btn-refresh-reviews">Refresh</button>
      </div>
      <div id="review-requests-error" class="error-banner"></div>
      ${prs.length ? prs.map(row).join("") : `<p class="empty">No PRs awaiting your review, and none reviewed recently.</p>`}
    `;
    document.getElementById("app").innerHTML = html;
    document.getElementById("btn-refresh-reviews").addEventListener("click", load);
  }

  async function load() {
    document.getElementById("app").innerHTML = `<p class="empty">Loading…</p>`;
    try {
      const prs = await Api.getReviewRequests();
      render(prs);
    } catch (e) {
      document.getElementById("app").innerHTML = `<div class="error-banner">Failed to load: ${Render.escapeHtml(e.message)}</div>`;
    }
  }

  return { load };
})();

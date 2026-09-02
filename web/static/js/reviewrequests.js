// "Review requested" view: a live, unstored list of PRs where you've been
// asked to review (or have already reviewed). Fetched fresh from GitHub
// every time this view loads — unlike the "Opened by me" dashboard, nothing
// here is tracked/persisted locally, except which PRs you've marked ignored.
const ReviewRequests = (() => {
  let lastPRs = []; // last-loaded list, so toggling ignore can re-render without a refetch

  // "Review Requested" and "Ignored" are two sides of the same toggle — click
  // either to flip it. "Reviewed" isn't interactive.
  function statusBadge(pr) {
    if (pr.status === "ignored") {
      return `<button type="button" class="badge gray badge-toggle" data-action="unignore"
                data-owner="${Render.escapeHtml(pr.owner)}" data-repo="${Render.escapeHtml(pr.repo)}" data-number="${pr.number}"
                title="Click to stop ignoring">Ignored</button>`;
    }
    if (pr.status === "review_requested") {
      return `<button type="button" class="badge amber badge-toggle" data-action="ignore"
                data-owner="${Render.escapeHtml(pr.owner)}" data-repo="${Render.escapeHtml(pr.repo)}" data-number="${pr.number}"
                title="Click if you're not going to review this">Review Requested</button>`;
    }
    return `<span class="badge green">Reviewed</span>`;
  }

  function row(pr) {
    return `
      <div class="pr-card">
        <div class="pr-main">
          <a class="pr-title" href="${pr.url}" target="_blank" rel="noopener">${Render.escapeHtml(pr.title)}</a>
          <div class="pr-meta">${Render.escapeHtml(pr.owner)}/${Render.escapeHtml(pr.repo)}#${pr.number}</div>
        </div>
        ${statusBadge(pr)}
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
    lastPRs = prs;
    const html = `
      ${statsBarHtml(prs)}
      <div class="toolbar">
        <button id="btn-refresh-reviews">Refresh</button>
      </div>
      <div id="review-requests-error" class="error-banner"></div>
      ${prs.length ? prs.map(row).join("") : `<p class="empty">No PRs awaiting your review, and none reviewed recently.</p>`}
    `;
    document.getElementById("app").innerHTML = html;
    wireUp();
  }

  function showError(msg) {
    const el = document.getElementById("review-requests-error");
    if (el) el.textContent = msg || "";
  }

  function wireUp() {
    document.getElementById("btn-refresh-reviews").addEventListener("click", load);

    document.querySelectorAll(".badge-toggle").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const { owner, repo, number, action } = btn.dataset;
        showError("");
        try {
          if (action === "ignore") {
            await Api.ignoreReviewRequest(owner, repo, number);
            // Ignoring always wins over the live state, so we can update in place with no refetch.
            setLocalStatus(owner, repo, Number(number), "ignored");
          } else {
            await Api.unignoreReviewRequest(owner, repo, number);
            // Un-ignoring needs a fresh look at GitHub to know the real current status.
            await load();
          }
        } catch (err) {
          showError(err.message);
        }
      });
    });
  }

  function setLocalStatus(owner, repo, number, status) {
    const pr = lastPRs.find((p) => p.owner === owner && p.repo === repo && p.number === number);
    if (pr) pr.status = status;
    render(lastPRs);
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

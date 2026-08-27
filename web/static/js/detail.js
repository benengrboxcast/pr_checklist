// Single-PR detail view: status bar, tag editor, review threads, and a
// read-only side panel of top-level (non-tracked) conversation comments.
const Detail = (() => {
  let current = null; // {owner, repo, number}

  function commentHtml(c) {
    return `
      <div class="comment">
        <div class="author">${Render.escapeHtml(c.author)} <span class="last-synced">${Render.formatDate(c.createdAt)}</span></div>
        <div class="body">${Render.escapeHtml(c.body)}</div>
      </div>`;
  }

  function threadHtml(t) {
    const canResolve = t.state === "unresolved";
    const canUnresolve = t.state === "resolved_locally";
    let actions = "";
    if (canResolve) {
      actions = `<button class="js-resolve" data-thread="${Render.escapeHtml(t.id)}">Mark resolved locally</button>`;
    } else if (canUnresolve) {
      actions = `<button class="js-unresolve" data-thread="${Render.escapeHtml(t.id)}">Undo local resolve</button>`;
    }
    return `
      <div class="thread">
        <div class="thread-header">
          <span class="badge ${threadBadgeColor(t.state)}">${Render.threadStateLabel(t.state)}</span>
          ${actions}
        </div>
        ${t.comments.map(commentHtml).join("")}
      </div>`;
  }

  function threadBadgeColor(state) {
    if (state === "resolved") return "green";
    if (state === "resolved_locally") return "amber";
    return "red";
  }

  function render(pr) {
    const counts = pr.threadCounts;
    const html = `
      <a class="back-link" href="#/">&larr; Back to dashboard</a>
      <h1>${Render.escapeHtml(pr.title)}</h1>
      <p class="pr-meta">
        <a href="${pr.url}" target="_blank" rel="noopener">${Render.escapeHtml(pr.owner)}/${Render.escapeHtml(pr.repo)}#${pr.number}</a>
        · ${Render.escapeHtml(pr.state)}
        · branch <code>${Render.escapeHtml(pr.headRefName)}</code>
        · by ${Render.escapeHtml(pr.authorLogin)}
      </p>

      <div id="detail-error" class="error-banner"></div>

      <div class="status-bar ${pr.status}">
        ${Render.statusBadge(pr.status)}
        <div class="counts">
          <span>Total: ${counts.total}</span>
          <span>Unresolved: ${counts.unresolved}</span>
          <span>Resolved locally: ${counts.resolvedLocally}</span>
          <span>Resolved: ${counts.resolved}</span>
        </div>
        <span class="last-synced">synced ${Render.formatDate(pr.lastSyncedAt)}</span>
        <button id="btn-refresh">Refresh</button>
      </div>

      <div style="margin-bottom:16px">
        ${Render.renderTags(pr.tags, { removable: true })}
        <form id="add-tag-form" style="display:inline-block; margin-left:8px">
          <input type="text" id="add-tag-input" placeholder="add tag" style="width:120px" />
          <button type="submit">+</button>
        </form>
      </div>

      <div class="layout">
        <div class="main">
          <h3>Review threads</h3>
          ${pr.reviewThreads.length ? pr.reviewThreads.map(threadHtml).join("") : `<p class="empty">No review threads.</p>`}
        </div>
        <div class="side">
          <div class="side-panel">
            <h3>Conversation (read-only)</h3>
            ${pr.comments.length ? pr.comments.map(commentHtml).join("") : `<p class="empty">No top-level comments.</p>`}
          </div>
        </div>
      </div>
    `;
    document.getElementById("app").innerHTML = html;
    wireUp(pr);
  }

  function showError(msg) {
    const el = document.getElementById("detail-error");
    if (el) el.textContent = msg || "";
  }

  async function load(owner, repo, number) {
    current = { owner, repo, number };
    try {
      const pr = await Api.getPR(owner, repo, number);
      render(pr);
    } catch (e) {
      document.getElementById("app").innerHTML = `<a class="back-link" href="#/">&larr; Back to dashboard</a><div class="error-banner">Failed to load: ${Render.escapeHtml(e.message)}</div>`;
    }
  }

  function wireUp(pr) {
    document.getElementById("btn-refresh").addEventListener("click", async () => {
      showError("");
      try {
        await Api.refreshPR(pr.owner, pr.repo, pr.number);
        await TagTitles.load(); // pick up any title auto-derived from this PR's branch name
        await load(pr.owner, pr.repo, pr.number);
      } catch (err) {
        showError(err.message);
      }
    });

    document.getElementById("add-tag-form").addEventListener("submit", async (e) => {
      e.preventDefault();
      const input = document.getElementById("add-tag-input");
      const tag = input.value.trim();
      if (!tag) return;
      showError("");
      try {
        await Api.addTag(pr.owner, pr.repo, pr.number, tag);
        await load(pr.owner, pr.repo, pr.number);
      } catch (err) {
        showError(err.message);
      }
    });

    document.querySelectorAll(".tag-remove").forEach((el) => {
      el.addEventListener("click", async (e) => {
        e.preventDefault();
        showError("");
        try {
          await Api.removeTag(pr.owner, pr.repo, pr.number, el.dataset.tag);
          await load(pr.owner, pr.repo, pr.number);
        } catch (err) {
          showError(err.message);
        }
      });
    });

    document.querySelectorAll(".js-resolve").forEach((btn) => {
      btn.addEventListener("click", async () => {
        showError("");
        try {
          await Api.resolveThread(pr.owner, pr.repo, pr.number, btn.dataset.thread);
          await load(pr.owner, pr.repo, pr.number);
        } catch (err) {
          showError(err.message);
        }
      });
    });

    document.querySelectorAll(".js-unresolve").forEach((btn) => {
      btn.addEventListener("click", async () => {
        showError("");
        try {
          await Api.unresolveThread(pr.owner, pr.repo, pr.number, btn.dataset.thread);
          await load(pr.owner, pr.repo, pr.number);
        } catch (err) {
          showError(err.message);
        }
      });
    });
  }

  return { load };
})();

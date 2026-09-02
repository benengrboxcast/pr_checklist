// Dashboard view: tag-grouped list of tracked PRs plus add/import/remove-closed/config actions.
const Dashboard = (() => {
  const FILTERS = {
    all: { label: "All", match: () => true },
    "not-approved": { label: "Not Approved", match: (pr) => pr.status !== "green" },
    "needs-attention": { label: "Needs Attention", match: (pr) => pr.status === "red" },
  };

  let currentFilter = "all";
  let lastPRs = []; // last full (unfiltered) list loaded, so switching filters doesn't need a refetch

  let bulkMode = false;
  let selectedKeys = new Set(); // "owner/repo/number" keys, since a multi-tagged PR can appear in several sections

  function keyOf(pr) {
    return `${pr.owner}/${pr.repo}/${pr.number}`;
  }

  function groupByTag(prs) {
    const groups = new Map();
    const untagged = [];
    for (const pr of prs) {
      if (!pr.tags || pr.tags.length === 0) {
        untagged.push(pr);
        continue;
      }
      for (const tag of pr.tags) {
        if (!groups.has(tag)) groups.set(tag, []);
        groups.get(tag).push(pr);
      }
    }
    const sortedTags = Array.from(groups.keys()).sort((a, b) => TagTitles.get(a).localeCompare(TagTitles.get(b)));
    return { groups, sortedTags, untagged };
  }

  function prCard(pr) {
    const link = `#/pr/${encodeURIComponent(pr.owner)}/${encodeURIComponent(pr.repo)}/${pr.number}`;
    const key = keyOf(pr);
    const checkbox = bulkMode
      ? `<input type="checkbox" class="bulk-select" data-key="${Render.escapeHtml(key)}" ${selectedKeys.has(key) ? "checked" : ""} />`
      : "";
    return `
      <div class="pr-card" data-owner="${Render.escapeHtml(pr.owner)}" data-repo="${Render.escapeHtml(pr.repo)}" data-number="${pr.number}">
        <div class="pr-labels-col">${Render.renderLabels(pr.labels, { onlyKnown: true })}</div>
        ${checkbox}
        <div class="pr-main">
          <a class="pr-title" href="${link}">${Render.escapeHtml(pr.title)}</a>
          <div class="pr-meta">
            ${Render.escapeHtml(pr.owner)}/${Render.escapeHtml(pr.repo)}#${pr.number}
            · ${pr.threadCounts.total} thread${pr.threadCounts.total === 1 ? "" : "s"}
            · synced ${Render.formatDate(pr.lastSyncedAt)}
            ${Render.renderTags(pr.tags)}
          </div>
        </div>
        ${Render.statusBadge(pr.status)}
        <div class="pr-actions">
          <button class="js-remove" title="Remove from dashboard">Remove</button>
        </div>
      </div>`;
  }

  // headingHtml is pre-rendered (e.g. via Render.renderTagTitleLink, which
  // already escapes and optionally Jira-links it) rather than raw text. The
  // section also gets its own badge aggregating every PR's status in it.
  function section(headingHtml, prs) {
    if (prs.length === 0) return "";
    const status = Render.groupStatus(prs);
    const badge = status ? Render.statusBadge(status) : "";
    return `<div class="tag-section"><div class="tag-section-header"><h2>${headingHtml}</h2>${badge}</div>${prs.map(prCard).join("")}</div>`;
  }

  // Stats always reflect the full, unfiltered list — the filter buttons only
  // narrow what's rendered below, they don't change what the counts mean.
  function statsBarHtml(prs) {
    const total = prs.length;
    const needsAttention = prs.filter((pr) => pr.status === "red").length;
    const inProgress = prs.filter((pr) => pr.status === "amber").length;
    const filterButtons = Object.entries(FILTERS)
      .map(
        ([key, f]) => `
        <button class="view-tab filter-btn${key === currentFilter ? " active" : ""}" data-filter="${key}" type="button">${f.label}</button>`
      )
      .join("");
    return `
      <div class="stats-bar">
        <span class="stat-total">${total} PR${total === 1 ? "" : "s"}</span>
        <span class="badge red">${needsAttention} Needs Attention</span>
        <span class="badge amber">${inProgress} In Progress</span>
        <div class="filter-group">${filterButtons}</div>
      </div>`;
  }

  function bulkBarHtml() {
    return `
      <div class="bulk-bar">
        <span id="bulk-count">${selectedKeys.size} selected</span>
        <button id="bulk-select-all" type="button">Select all</button>
        <button id="bulk-clear-selection" type="button">Clear</button>
        <input type="text" id="bulk-tag-input" placeholder="tag to apply" />
        <button id="bulk-apply" class="primary" type="button">Apply tag</button>
      </div>`;
  }

  function render(prs) {
    lastPRs = prs;
    const filtered = prs.filter(FILTERS[currentFilter].match);
    const { groups, sortedTags, untagged } = groupByTag(filtered);
    let html = `
      ${statsBarHtml(prs)}
      <div class="toolbar">
        <form id="add-pr-form">
          <input type="url" id="add-pr-url" placeholder="https://github.com/owner/repo/pull/123" required />
          <button type="submit" class="primary">Add PR</button>
        </form>
        <button id="btn-import">Import my open PRs</button>
        <button id="btn-remove-closed" class="danger">Remove closed</button>
        <button id="btn-bulk-toggle" class="${bulkMode ? "active" : ""}">${bulkMode ? "Exit bulk edit" : "Bulk edit"}</button>
        <button id="btn-settings">Settings</button>
      </div>
      ${bulkMode ? bulkBarHtml() : ""}
      <div id="dashboard-error" class="error-banner"></div>
      <div id="settings-panel" style="display:none"></div>
    `;

    if (prs.length === 0) {
      html += `<p class="empty">No PRs tracked yet. Paste a PR URL above, or use Import.</p>`;
    } else if (filtered.length === 0) {
      html += `<p class="empty">No PRs match the "${FILTERS[currentFilter].label}" filter.</p>`;
    } else {
      for (const tag of sortedTags) html += section(Render.renderTagTitleLink(tag), groups.get(tag));
      html += section(Render.escapeHtml("Untagged"), untagged);
    }

    document.getElementById("app").innerHTML = html;
    wireUp();
  }

  function showError(msg) {
    document.getElementById("dashboard-error").textContent = msg || "";
  }

  async function reload() {
    try {
      const prs = await Api.listPRs();
      render(prs);
    } catch (e) {
      document.getElementById("app").innerHTML = `<div class="error-banner">Failed to load: ${Render.escapeHtml(e.message)}</div>`;
    }
  }

  // Updates just the "N selected" count in place, without a full re-render,
  // so ticking checkboxes stays snappy while selecting many PRs.
  function updateBulkCount() {
    const el = document.getElementById("bulk-count");
    if (el) el.textContent = `${selectedKeys.size} selected`;
  }

  function wireUp() {
    document.querySelectorAll(".filter-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        currentFilter = btn.dataset.filter;
        render(lastPRs); // re-render from the already-loaded list, no refetch needed
      });
    });

    document.getElementById("add-pr-form").addEventListener("submit", async (e) => {
      e.preventDefault();
      const input = document.getElementById("add-pr-url");
      showError("");
      try {
        await Api.addPR(input.value.trim());
        await TagTitles.load(); // pick up any title auto-derived from this PR's branch name
        await reload();
      } catch (err) {
        showError(err.message);
      }
    });

    document.getElementById("btn-import").addEventListener("click", async () => {
      showError("");
      try {
        const result = await Api.importPRs();
        await TagTitles.load(); // pick up any titles auto-derived from imported PRs' branch names
        await reload();
        if (result.errors && result.errors.length > 0) {
          showError(`Import finished with errors: ${result.errors.join("; ")}`);
        }
      } catch (err) {
        showError(err.message);
      }
    });

    document.getElementById("btn-remove-closed").addEventListener("click", async () => {
      if (!confirm("Remove every tracked PR that is closed or merged?")) return;
      showError("");
      try {
        await Api.bulkRemoveClosed();
        await reload();
      } catch (err) {
        showError(err.message);
      }
    });

    document.getElementById("btn-settings").addEventListener("click", Settings.toggle);

    document.getElementById("btn-bulk-toggle").addEventListener("click", () => {
      bulkMode = !bulkMode;
      selectedKeys.clear();
      render(lastPRs);
    });

    document.querySelectorAll(".js-remove").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const card = btn.closest(".pr-card");
        const { owner, repo, number } = card.dataset;
        if (!confirm(`Remove ${owner}/${repo}#${number} from the dashboard? This permanently deletes its local data.`)) return;
        showError("");
        try {
          await Api.deletePR(owner, repo, Number(number));
          await reload();
        } catch (err) {
          showError(err.message);
        }
      });
    });

    if (!bulkMode) return;

    // A PR with several tags renders once per tag section, so keep every
    // checkbox for the same PR in sync with each other.
    document.querySelectorAll(".bulk-select").forEach((cb) => {
      cb.addEventListener("change", () => {
        const key = cb.dataset.key;
        if (cb.checked) selectedKeys.add(key);
        else selectedKeys.delete(key);
        document.querySelectorAll(`.bulk-select[data-key="${CSS.escape(key)}"]`).forEach((other) => {
          other.checked = cb.checked;
        });
        updateBulkCount();
      });
    });

    document.getElementById("bulk-select-all").addEventListener("click", () => {
      document.querySelectorAll(".bulk-select").forEach((cb) => {
        cb.checked = true;
        selectedKeys.add(cb.dataset.key);
      });
      updateBulkCount();
    });

    document.getElementById("bulk-clear-selection").addEventListener("click", () => {
      selectedKeys.clear();
      document.querySelectorAll(".bulk-select").forEach((cb) => (cb.checked = false));
      updateBulkCount();
    });

    document.getElementById("bulk-apply").addEventListener("click", async () => {
      const tag = document.getElementById("bulk-tag-input").value.trim();
      if (!tag) {
        showError("Enter a tag to apply.");
        return;
      }
      if (selectedKeys.size === 0) {
        showError("Select at least one PR first.");
        return;
      }
      const prs = Array.from(selectedKeys).map((key) => {
        const [owner, repo, number] = key.split("/");
        return { owner, repo, number: Number(number) };
      });
      showError("");
      try {
        const result = await Api.bulkTag(prs, tag);
        await TagTitles.load();
        bulkMode = false;
        selectedKeys.clear();
        await reload();
        if (result.errors && result.errors.length > 0) {
          showError(`Tagged ${result.updated} PR${result.updated === 1 ? "" : "s"}, with errors: ${result.errors.join("; ")}`);
        }
      } catch (err) {
        showError(err.message);
      }
    });
  }

  return { reload, showError };
})();

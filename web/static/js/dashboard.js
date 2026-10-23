// Dashboard view: tag-grouped list of tracked PRs plus add/import/remove-closed/config actions.
const Dashboard = (() => {
  const FILTERS = {
    all: { label: "All", match: () => true },
    "not-approved": { label: "Not Approved", match: (pr) => pr.status !== "green" },
    "needs-attention": { label: "Needs Attention", match: (pr) => pr.status === "red" },
  };

  const GROUP_MODES = {
    tag: { label: "By Tag" },
    repo: { label: "By Repo" },
  };

  let currentFilter = "all";
  let groupMode = "tag";
  let lastPRs = []; // last full (unfiltered) list loaded, so switching filters doesn't need a refetch
  let listableSections = []; // {heading, prs} per section rendered with a "list prs" link, rebuilt on every render

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

  // Groups every PR by its actual GitHub repo (owner/repo), one section per
  // repo actually present, sorted by "owner/repo". Distinct from the
  // user-defined named repo groups in Settings (groupByRepoGroup below) —
  // this is the "By Repo" view toggle, automatic and covering every repo.
  function groupByRepo(prs) {
    const map = new Map();
    for (const pr of prs) {
      const key = `${pr.owner}/${pr.repo}`;
      if (!map.has(key)) map.set(key, { owner: pr.owner, repo: pr.repo, prs: [] });
      map.get(key).prs.push(pr);
    }
    return Array.from(map.values()).sort((a, b) => `${a.owner}/${a.repo}`.localeCompare(`${b.owner}/${b.repo}`));
  }

  function repoHeadingHtml(owner, repo) {
    const url = `https://github.com/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/pulls`;
    return `<a href="${url}" target="_blank" rel="noopener">${Render.escapeHtml(owner)}/${Render.escapeHtml(repo)}</a>`;
  }

  // Matches each user-defined repo group (Settings) against prs, case-
  // insensitively on "owner/repo". Sorted by group name, and always placed
  // at the end of the dashboard, after every tag section and Untagged.
  function groupByRepoGroup(prs) {
    return RepoGroups.get()
      .map((g) => {
        const repoSet = new Set((g.repos || []).map((r) => Render.normalizeRepoSlug(r).toLowerCase()).filter(Boolean));
        const matched = prs.filter((pr) => repoSet.has(`${pr.owner}/${pr.repo}`.toLowerCase()));
        return { name: g.name, prs: matched };
      })
      .sort((a, b) => a.name.localeCompare(b.name));
  }

  // headingHtml is pre-rendered (e.g. via Render.renderTagTitleLink, which
  // already escapes and optionally Jira-links it) rather than raw text. The
  // section also gets its own badge aggregating every PR's status in it.
  //
  // With opts.listable (tag sections), the header also gets a PR-count badge
  // and a right-aligned "list prs" link that opens a copyable list of the
  // section's PR URLs; the section's PRs are stashed in listableSections so
  // the click handler can look them up by index.
  function section(headingHtml, prs, opts = {}) {
    if (prs.length === 0) return "";
    const status = Render.groupStatus(prs);
    const badge = status ? Render.statusBadge(status) : "";
    let listing = "";
    if (opts.listable) {
      const idx = listableSections.push({ heading: opts.listTitle, prs }) - 1;
      listing = `<span class="badge count" title="PRs in this section">${prs.length}</span>
        <a href="#" class="list-prs-link" data-section="${idx}">list prs</a>`;
    }
    return `<div class="tag-section"><div class="tag-section-header"><h2>${headingHtml}</h2>${badge}${listing}</div>${prs.map(prCard).join("")}</div>`;
  }

  function prUrl(pr) {
    return `https://github.com/${pr.owner}/${pr.repo}/pull/${pr.number}`;
  }

  // Shows a modal with one PR URL per line in a readonly textarea, ready to
  // paste into Slack (which auto-links bare URLs), plus a Copy button.
  function showPRListDialog(heading, prs) {
    const text = prs.map(prUrl).join("\n");
    const dlg = document.createElement("dialog");
    dlg.className = "pr-list-dialog";
    dlg.innerHTML = `
      <h3>${Render.escapeHtml(heading)} · ${prs.length} PR${prs.length === 1 ? "" : "s"}</h3>
      <textarea readonly rows="${Math.min(Math.max(prs.length, 3), 15)}"></textarea>
      <div class="pr-list-dialog-actions">
        <span class="pr-list-copied"></span>
        <button type="button" class="primary js-copy">Copy</button>
        <button type="button" class="js-close">Close</button>
      </div>`;
    const ta = dlg.querySelector("textarea");
    ta.value = text;
    dlg.querySelector(".js-close").addEventListener("click", () => dlg.close());
    dlg.querySelector(".js-copy").addEventListener("click", async () => {
      const note = dlg.querySelector(".pr-list-copied");
      try {
        await navigator.clipboard.writeText(text);
      } catch (_) {
        ta.select(); // clipboard API unavailable (e.g. non-secure origin); fall back to the legacy command
        document.execCommand("copy");
      }
      note.textContent = "Copied!";
    });
    dlg.addEventListener("click", (e) => {
      if (e.target === dlg) dlg.close(); // backdrop click
    });
    dlg.addEventListener("close", () => dlg.remove());
    document.body.appendChild(dlg);
    dlg.showModal();
    ta.select();
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
    const groupButtons = Object.entries(GROUP_MODES)
      .map(
        ([key, g]) => `
        <button class="view-tab filter-btn group-mode-btn${key === groupMode ? " active" : ""}" data-group="${key}" type="button">${g.label}</button>`
      )
      .join("");
    return `
      <div class="stats-bar">
        <span class="stat-total">${total} PR${total === 1 ? "" : "s"}</span>
        <span class="badge red">${needsAttention} Needs Attention</span>
        <span class="badge amber">${inProgress} In Progress</span>
        <div class="filter-group">${filterButtons}</div>
        <div class="filter-group">${groupButtons}</div>
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
    listableSections = [];
    const filtered = prs.filter(FILTERS[currentFilter].match);
    const { groups, sortedTags, untagged } = groupByTag(filtered);
    let html = `
      ${statsBarHtml(prs)}
      <div class="toolbar">
        <form id="add-pr-form">
          <input type="url" id="add-pr-url" placeholder="https://github.com/owner/repo/pull/123" required />
          <button type="submit" class="primary">Add PR</button>
        </form>
        <button id="btn-update">Update</button>
        <button id="btn-remove-closed" class="danger">Remove closed</button>
        <button id="btn-bulk-toggle" class="${bulkMode ? "active" : ""}">${bulkMode ? "Exit bulk edit" : "Bulk edit"}</button>
        <button id="btn-settings">Settings</button>
      </div>
      ${bulkMode ? bulkBarHtml() : ""}
      <div id="dashboard-error" class="error-banner"></div>
      <div id="settings-panel" style="display:none"></div>
    `;

    if (prs.length === 0) {
      html += `<p class="empty">No PRs tracked yet. Paste a PR URL above, or click Update to pull in your open PRs.</p>`;
    } else if (filtered.length === 0) {
      html += `<p class="empty">No PRs match the "${FILTERS[currentFilter].label}" filter.</p>`;
    } else if (groupMode === "repo") {
      for (const g of groupByRepo(filtered)) html += section(repoHeadingHtml(g.owner, g.repo), g.prs);
    } else {
      for (const tag of sortedTags) {
        html += section(Render.renderTagTitleLink(tag), groups.get(tag), { listable: true, listTitle: TagTitles.get(tag) });
      }
      html += section(Render.escapeHtml("Untagged"), untagged);
      for (const g of groupByRepoGroup(filtered)) html += section(Render.escapeHtml(g.name), g.prs);
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

  // Grows the "Updating…" button's fill bar to pct%, animated via CSS transition.
  function setUpdateProgress(pct) {
    const fill = document.getElementById("btn-update-fill");
    if (fill) fill.style.width = `${pct}%`;
  }

  function wireUp() {
    document.querySelectorAll(".filter-btn:not(.group-mode-btn)").forEach((btn) => {
      btn.addEventListener("click", () => {
        currentFilter = btn.dataset.filter;
        render(lastPRs); // re-render from the already-loaded list, no refetch needed
      });
    });

    document.querySelectorAll(".group-mode-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        groupMode = btn.dataset.group;
        render(lastPRs);
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

    document.getElementById("btn-update").addEventListener("click", async () => {
      showError("");
      const btn = document.getElementById("btn-update");
      btn.disabled = true;
      // The fill starts at a non-zero width set directly in its initial markup
      // (rather than 0% + a rAF-driven bump) so it still shows progress even
      // if the tab is backgrounded during the update — requestAnimationFrame
      // callbacks are throttled/paused in background tabs, but plain style
      // changes tied to promise resolutions below aren't.
      btn.innerHTML = `<span class="btn-progress-fill" id="btn-update-fill" style="width:8%"></span><span class="btn-label">Updating…</span>`;
      try {
        // Refresh every already-tracked PR's state, then pull in any newly
        // opened PRs — in that order so a freshly-imported PR isn't
        // redundantly re-fetched a second time by refresh-all. The bar's two
        // steps mirror these two real phases, not a fake timer.
        await Api.refreshAll();
        setUpdateProgress(55);
        const result = await Api.importPRs();
        setUpdateProgress(100);
        await TagTitles.load(); // pick up any titles auto-derived from refreshed/imported PRs' branch names
        await reload();
        if (result.errors && result.errors.length > 0) {
          showError(`Update finished with errors: ${result.errors.join("; ")}`);
        }
      } catch (err) {
        showError(err.message);
      } finally {
        const stillThere = document.getElementById("btn-update");
        if (stillThere) {
          stillThere.disabled = false;
          stillThere.textContent = "Update";
        }
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

    document.querySelectorAll(".list-prs-link").forEach((a) => {
      a.addEventListener("click", (e) => {
        e.preventDefault();
        const s = listableSections[Number(a.dataset.section)];
        if (s) showPRListDialog(s.heading, s.prs);
      });
    });

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

// Settings panel: org checklist + repo allow-list for Import, and per-tag
// display titles (with an auto-generate action). Lives on the dashboard view.
const Settings = (() => {
  let lastConfig = {}; // stashed so save() can pass through fields this panel doesn't edit (e.g. ignoredReviewPRs)

  function panelEl() {
    return document.getElementById("settings-panel");
  }

  function isOpen() {
    const panel = panelEl();
    return !!panel && panel.style.display !== "none";
  }

  function close() {
    const panel = panelEl();
    panel.style.display = "none";
    panel.innerHTML = "";
  }

  async function toggle() {
    if (isOpen()) {
      close();
      return;
    }
    const panel = panelEl();
    panel.style.display = "block";
    await render();
  }

  async function render() {
    const panel = panelEl();
    panel.innerHTML = `<div class="side-panel">Loading settings…</div>`;

    let cfg, prs;
    try {
      [cfg, prs] = await Promise.all([Api.getConfig(), Api.listPRs()]);
      lastConfig = cfg;
    } catch (err) {
      Dashboard.showError(err.message);
      close();
      return;
    }

    let orgs = [];
    let orgsError = "";
    try {
      const result = await Api.getOrgs();
      orgs = result.orgs || [];
    } catch (err) {
      orgsError = err.message;
    }

    panel.innerHTML = buildHtml(cfg, prs, orgs, orgsError);
    wireUp();
  }

  function buildHtml(cfg, prs, orgs, orgsError) {
    const selectedOrgs = new Set(cfg.importOrgs || []);
    const orgsHtml = orgs.length
      ? orgs
          .map(
            (org) => `
        <label style="display:block; margin-bottom:4px">
          <input type="checkbox" class="org-checkbox" value="${Render.escapeHtml(org)}" ${selectedOrgs.has(org) ? "checked" : ""} />
          ${Render.escapeHtml(org)}
        </label>`
          )
          .join("")
      : `<p class="last-synced">${
          orgsError ? `Could not load orgs: ${Render.escapeHtml(orgsError)}` : "No organizations found for this account."
        }</p>`;

    const tagTitles = cfg.tagTitles || {};
    const allTags = new Set([...Object.keys(tagTitles), ...prs.flatMap((pr) => pr.tags || [])]);
    const sortedTags = Array.from(allTags).sort((a, b) => a.localeCompare(b));
    const tagTitlesHtml = sortedTags.length
      ? sortedTags
          .map(
            (tag) => `
        <div style="display:flex; gap:8px; align-items:center; margin-bottom:4px">
          <span style="min-width:140px; font-size:13px">${Render.renderTagLink(tag)}</span>
          <input type="text" class="tag-title-input" data-tag="${Render.escapeHtml(tag)}"
                 value="${Render.escapeHtml(tagTitles[tag] || "")}" placeholder="(shown as raw label)" style="flex:1" />
        </div>`
          )
          .join("")
      : `<p class="last-synced">No labels yet.</p>`;

    return `
      <div class="side-panel">
        <h3>Import from orgs</h3>
        <p class="last-synced">Check an org to import every one of your open PRs across all its repos.</p>
        ${orgsHtml}

        <h3 style="margin-top:16px">Import repo allow-list</h3>
        <p class="last-synced">One "owner/repo" per line, for repos outside any checked org above.</p>
        <textarea id="import-repos" rows="4" style="width:100%">${Render.escapeHtml((cfg.importRepos || []).join("\n"))}</textarea>

        <h3 style="margin-top:16px">Description tagging</h3>
        <p class="last-synced">
          Scans every tracked PR's stored description for a "Relevant Jira issue(s):" section and tags any issue
          keys found there. Purely additive, never removes a tag. Only sees descriptions already fetched — refresh
          a PR first if its description changed since it was last synced.
        </p>
        <div style="margin-top:8px">
          <button id="auto-tag-descriptions" type="button">Auto-tag from descriptions</button>
        </div>
        <div id="auto-tag-descriptions-status" class="last-synced" style="margin-top:4px"></div>

        <h3 style="margin-top:16px">Label titles</h3>
        <p class="last-synced">
          Custom display text shown instead of a raw label (e.g. EMBED-2575 &rarr; "Start Stop Extend").
          Leave blank to show the raw label. Hover a label chip anywhere to see its raw value.
        </p>
        <div id="tag-titles-list">${tagTitlesHtml}</div>
        <div style="margin-top:8px">
          <button id="auto-generate-titles" type="button">Auto-generate from branch names</button>
        </div>
        <div id="auto-generate-status" class="last-synced" style="margin-top:4px"></div>

        <div style="margin-top:16px"><button id="save-config" class="primary">Save</button></div>
      </div>`;
  }

  function wireUp() {
    document.getElementById("save-config").addEventListener("click", save);
    document.getElementById("auto-generate-titles").addEventListener("click", autoGenerate);
    document.getElementById("auto-tag-descriptions").addEventListener("click", autoTagFromDescriptions);
  }

  async function save() {
    const repos = document
      .getElementById("import-repos")
      .value.split("\n")
      .map((s) => s.trim())
      .filter(Boolean);
    const selectedOrgs = Array.from(document.querySelectorAll(".org-checkbox:checked")).map((el) => el.value);
    const tagTitles = {};
    document.querySelectorAll(".tag-title-input").forEach((el) => {
      const v = el.value.trim();
      if (v) tagTitles[el.dataset.tag] = v;
    });

    Dashboard.showError("");
    try {
      await Api.putConfig(repos, selectedOrgs, tagTitles, lastConfig.ignoredReviewPRs || []);
      await TagTitles.load();
      close();
      await Dashboard.reload();
    } catch (err) {
      Dashboard.showError(err.message);
    }
  }

  async function autoGenerate() {
    const statusBefore = document.getElementById("auto-generate-status");
    if (statusBefore) statusBefore.textContent = "Generating…";
    try {
      const result = await Api.autoGenerateTagTitles();
      await TagTitles.load();
      await render(); // re-render in place with the newly-filled titles, panel stays open
      const status = document.getElementById("auto-generate-status");
      if (status) status.textContent = `Added ${result.added} title${result.added === 1 ? "" : "s"}.`;
    } catch (err) {
      Dashboard.showError(err.message);
    }
  }

  async function autoTagFromDescriptions() {
    const statusBefore = document.getElementById("auto-tag-descriptions-status");
    if (statusBefore) statusBefore.textContent = "Scanning descriptions…";
    try {
      const result = await Api.autoTagFromDescriptions();
      await TagTitles.load();
      await render(); // re-render in place; newly-found tags may show up in the Label titles list
      const status = document.getElementById("auto-tag-descriptions-status");
      if (status) {
        status.textContent = `Added ${result.added} tag${result.added === 1 ? "" : "s"}.`;
        if (result.errors && result.errors.length > 0) {
          status.textContent += ` (${result.errors.length} error(s) — see console)`;
          console.error("auto-tag-from-descriptions errors:", result.errors);
        }
      }
    } catch (err) {
      Dashboard.showError(err.message);
    }
  }

  return { toggle };
})();

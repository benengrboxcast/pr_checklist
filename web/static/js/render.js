// Shared rendering helpers used by both the dashboard and detail views.
const Render = (() => {
  const JIRA_TAG_RE = /^[A-Z]{2,10}-\d+$/;

  function escapeHtml(str) {
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function statusLabel(status) {
    return { gray: "Closed", red: "Needs Attention", green: "Approved", amber: "In Progress" }[status] || status;
  }

  function statusBadge(status) {
    return `<span class="badge ${status}">${statusLabel(status)}</span>`;
  }

  // Aggregates a set of PRs' individual statuses into one badge status for a
  // group (e.g. a tag section): red if any PR needs attention, else amber if
  // any is still in progress, else green only if every PR is approved.
  // Returns null if none of those conditions hold (e.g. an all-closed group).
  function groupStatus(prs) {
    if (!prs || prs.length === 0) return null;
    if (prs.some((pr) => pr.status === "red")) return "red";
    if (prs.some((pr) => pr.status === "amber")) return "amber";
    if (prs.every((pr) => pr.status === "green")) return "green";
    return null;
  }

  function threadStateLabel(state) {
    return { unresolved: "Unresolved", resolved_locally: "Resolved locally", resolved: "Resolved" }[state] || state;
  }

  // Renders a tag chip, showing its custom title (if set) instead of the raw
  // tag text — hover the chip to see the raw tag. Jira-shaped tags (e.g.
  // "EMBED-1234") link out to the Jira issue (keyed off the raw tag,
  // regardless of what title is shown), whether the tag was auto-detected
  // from a branch/description or typed by hand.
  function renderTag(tag, opts) {
    opts = opts || {};
    const raw = escapeHtml(tag);
    const display = escapeHtml(TagTitles.get(tag));
    const removable = opts.removable ? ` <a href="#" class="tag-remove" data-tag="${raw}" title="Remove tag">×</a>` : "";
    if (JIRA_TAG_RE.test(tag)) {
      return `<a class="tag-chip" href="https://boxcast.atlassian.net/browse/${raw}" target="_blank" rel="noopener" title="${raw}">${display}</a>${removable}`;
    }
    return `<span class="tag-chip" title="${raw}">${display}</span>${removable}`;
  }

  function renderTags(tags, opts) {
    if (!tags || tags.length === 0) return "";
    return tags.map((t) => renderTag(t, opts)).join(" ");
  }

  // Renders just the raw tag identifier as a Jira link when it's a
  // Jira-shaped tag, plain text otherwise. For contexts (like the Settings
  // label-titles editor) that show the raw tag itself rather than a full
  // styled chip.
  function renderTagLink(tag) {
    const raw = escapeHtml(tag);
    if (JIRA_TAG_RE.test(tag)) {
      return `<a href="https://boxcast.atlassian.net/browse/${raw}" target="_blank" rel="noopener">${raw}</a>`;
    }
    return raw;
  }

  // Renders a tag's display title (not the raw tag) as a Jira link when it's
  // a Jira-shaped tag, plain text otherwise. For heading-style contexts (like
  // dashboard tag-section headers) that show the title without chip styling.
  function renderTagTitleLink(tag) {
    const raw = escapeHtml(tag);
    const display = escapeHtml(TagTitles.get(tag));
    if (JIRA_TAG_RE.test(tag)) {
      return `<a href="https://boxcast.atlassian.net/browse/${raw}" target="_blank" rel="noopener" title="${raw}">${display}</a>`;
    }
    return display;
  }

  // The subset of GitHub PR labels our workflow cares about. The dashboard
  // only surfaces these three (to avoid cluttering PR cards with every label
  // a repo happens to use); the detail page shows all of a PR's labels.
  const WORKFLOW_LABELS = ["in-review", "in-dev", "verified-in-dev"];

  // Slugifies a label name into a CSS class suffix, e.g. "verified-in-dev" ->
  // "label-verified-in-dev". The three WORKFLOW_LABELS have dedicated color
  // rules in style.css; any other label just falls back to .label-chip's
  // neutral outline style since no matching rule exists for its slug.
  function labelClass(name) {
    const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, "-");
    return `label-chip label-${slug}`;
  }

  // Renders GitHub's own PR labels (read-only — distinct from this app's
  // editable tags) as chips. Pass onlyKnown:true to show only the
  // WORKFLOW_LABELS subset (used on the dashboard).
  function renderLabels(labels, opts) {
    opts = opts || {};
    if (!labels || labels.length === 0) return "";
    const shown = opts.onlyKnown ? labels.filter((l) => WORKFLOW_LABELS.includes(l)) : labels;
    if (shown.length === 0) return "";
    return shown.map((l) => `<span class="${labelClass(l)}">${escapeHtml(l)}</span>`).join(" ");
  }

  function formatDate(iso) {
    if (!iso) return "never";
    const d = new Date(iso);
    if (isNaN(d.getTime())) return "never";
    return d.toLocaleString();
  }

  return {
    escapeHtml,
    statusBadge,
    statusLabel,
    groupStatus,
    threadStateLabel,
    renderTag,
    renderTags,
    renderTagLink,
    renderTagTitleLink,
    renderLabels,
    formatDate,
  };
})();

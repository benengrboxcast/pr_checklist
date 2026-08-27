// Shared rendering helpers used by both the dashboard and detail views.
const Render = (() => {
  const EMBED_TAG_RE = /^EMBED-\d+$/;

  function escapeHtml(str) {
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function statusLabel(status) {
    return { gray: "Closed", red: "Needs attention", green: "All resolved & approved", amber: "In progress" }[status] || status;
  }

  function statusBadge(status) {
    return `<span class="badge ${status}">${statusLabel(status)}</span>`;
  }

  function threadStateLabel(state) {
    return { unresolved: "Unresolved", resolved_locally: "Resolved locally", resolved: "Resolved" }[state] || state;
  }

  // Renders a tag chip. EMBED-#### tags link out to the Jira issue,
  // whether the tag was auto-detected from a branch name or typed by hand.
  function renderTag(tag, opts) {
    opts = opts || {};
    const label = escapeHtml(tag);
    const removable = opts.removable ? ` <a href="#" class="tag-remove" data-tag="${label}" title="Remove tag">×</a>` : "";
    if (EMBED_TAG_RE.test(tag)) {
      return `<a class="tag-chip" href="https://boxcast.atlassian.net/browse/${label}" target="_blank" rel="noopener">${label}</a>${removable}`;
    }
    return `<span class="tag-chip">${label}</span>${removable}`;
  }

  function renderTags(tags, opts) {
    if (!tags || tags.length === 0) return "";
    return tags.map((t) => renderTag(t, opts)).join(" ");
  }

  function formatDate(iso) {
    if (!iso) return "never";
    const d = new Date(iso);
    if (isNaN(d.getTime())) return "never";
    return d.toLocaleString();
  }

  return { escapeHtml, statusBadge, statusLabel, threadStateLabel, renderTag, renderTags, formatDate };
})();

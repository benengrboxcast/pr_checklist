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
  // tag text — hover the chip to see the raw tag. EMBED-#### tags link out to
  // the Jira issue (keyed off the raw tag, regardless of what title is shown),
  // whether the tag was auto-detected from a branch name or typed by hand.
  function renderTag(tag, opts) {
    opts = opts || {};
    const raw = escapeHtml(tag);
    const display = escapeHtml(TagTitles.get(tag));
    const removable = opts.removable ? ` <a href="#" class="tag-remove" data-tag="${raw}" title="Remove tag">×</a>` : "";
    if (EMBED_TAG_RE.test(tag)) {
      return `<a class="tag-chip" href="https://boxcast.atlassian.net/browse/${raw}" target="_blank" rel="noopener" title="${raw}">${display}</a>${removable}`;
    }
    return `<span class="tag-chip" title="${raw}">${display}</span>${removable}`;
  }

  function renderTags(tags, opts) {
    if (!tags || tags.length === 0) return "";
    return tags.map((t) => renderTag(t, opts)).join(" ");
  }

  // Renders just the raw tag identifier as a Jira link when it's an EMBED-####
  // tag, plain text otherwise. For contexts (like the Settings label-titles
  // editor) that show the raw tag itself rather than a full styled chip.
  function renderTagLink(tag) {
    const raw = escapeHtml(tag);
    if (EMBED_TAG_RE.test(tag)) {
      return `<a href="https://boxcast.atlassian.net/browse/${raw}" target="_blank" rel="noopener">${raw}</a>`;
    }
    return raw;
  }

  // Renders a tag's display title (not the raw tag) as a Jira link when it's
  // an EMBED-#### tag, plain text otherwise. For heading-style contexts (like
  // dashboard tag-section headers) that show the title without chip styling.
  function renderTagTitleLink(tag) {
    const raw = escapeHtml(tag);
    const display = escapeHtml(TagTitles.get(tag));
    if (EMBED_TAG_RE.test(tag)) {
      return `<a href="https://boxcast.atlassian.net/browse/${raw}" target="_blank" rel="noopener" title="${raw}">${display}</a>`;
    }
    return display;
  }

  function formatDate(iso) {
    if (!iso) return "never";
    const d = new Date(iso);
    if (isNaN(d.getTime())) return "never";
    return d.toLocaleString();
  }

  return { escapeHtml, statusBadge, statusLabel, groupStatus, threadStateLabel, renderTag, renderTags, renderTagLink, renderTagTitleLink, formatDate };
})();

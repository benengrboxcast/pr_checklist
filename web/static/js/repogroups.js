// Cache of user-defined repo groups (name + list of "owner/repo" entries),
// loaded from server config. The dashboard uses this to render a section per
// group at the end of the page, alongside its usual tag sections.
const RepoGroups = (() => {
  let cache = [];

  async function load() {
    try {
      const cfg = await Api.getConfig();
      cache = cfg.repoGroups || [];
    } catch (_) {
      // leave cache as-is; dashboard just won't show repo-group sections until it succeeds
    }
  }

  function get() {
    return cache;
  }

  return { load, get };
})();

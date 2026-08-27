// Cache of tag -> custom display title (e.g. "EMBED-2575" -> "Start Stop Extend"),
// loaded from server config. Chips fall back to the raw tag if a title isn't set.
const TagTitles = (() => {
  let cache = {};

  async function load() {
    try {
      const cfg = await Api.getConfig();
      cache = cfg.tagTitles || {};
    } catch (_) {
      // leave cache as-is; chips just fall back to raw tag text
    }
  }

  function get(tag) {
    return cache[tag] || tag;
  }

  return { load, get };
})();

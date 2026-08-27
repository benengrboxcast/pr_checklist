// Thin fetch wrappers, one per backend endpoint.
const Api = (() => {
  async function request(method, path, body) {
    const res = await fetch(path, {
      method,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      let message = res.statusText;
      try {
        const data = await res.json();
        if (data && data.error) message = data.error;
      } catch (_) {}
      throw new Error(message);
    }
    if (res.status === 204) return null;
    return res.json();
  }

  const prPath = (owner, repo, number) =>
    `/api/prs/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/${number}`;

  return {
    listPRs: () => request("GET", "/api/prs"),
    getPR: (owner, repo, number) => request("GET", prPath(owner, repo, number)),
    addPR: (url) => request("POST", "/api/prs", { url }),
    deletePR: (owner, repo, number) => request("DELETE", prPath(owner, repo, number)),
    bulkRemoveClosed: () => request("POST", "/api/prs/bulk-remove-closed"),
    refreshPR: (owner, repo, number) => request("POST", `${prPath(owner, repo, number)}/refresh`),
    refreshAll: () => request("POST", "/api/prs/refresh-all"),
    importPRs: () => request("POST", "/api/import"),
    resolveThread: (owner, repo, number, threadId) =>
      request("POST", `${prPath(owner, repo, number)}/threads/${encodeURIComponent(threadId)}/resolve`),
    unresolveThread: (owner, repo, number, threadId) =>
      request("POST", `${prPath(owner, repo, number)}/threads/${encodeURIComponent(threadId)}/unresolve`),
    addTag: (owner, repo, number, tag) => request("POST", `${prPath(owner, repo, number)}/tags`, { tag }),
    removeTag: (owner, repo, number, tag) =>
      request("DELETE", `${prPath(owner, repo, number)}/tags/${encodeURIComponent(tag)}`),
    getConfig: () => request("GET", "/api/config"),
    putConfig: (importRepos, importOrgs, tagTitles) => request("PUT", "/api/config", { importRepos, importOrgs, tagTitles }),
    autoGenerateTagTitles: () => request("POST", "/api/config/auto-generate-titles"),
    getViewer: () => request("GET", "/api/viewer"),
    getOrgs: () => request("GET", "/api/orgs"),
  };
})();

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
  const reviewRequestPath = (owner, repo, number) =>
    `/api/review-requests/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/${number}`;

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
    bulkTag: (prs, tag) => request("POST", "/api/prs/bulk-tag", { prs, tag }),
    removeTag: (owner, repo, number, tag) =>
      request("DELETE", `${prPath(owner, repo, number)}/tags/${encodeURIComponent(tag)}`),
    tagFromDescription: (owner, repo, number) => request("POST", `${prPath(owner, repo, number)}/tag-from-description`),
    autoTagFromDescriptions: () => request("POST", "/api/prs/auto-tag-from-descriptions"),
    getConfig: () => request("GET", "/api/config"),
    putConfig: (importRepos, importOrgs, tagTitles, ignoredReviewPRs) =>
      request("PUT", "/api/config", { importRepos, importOrgs, tagTitles, ignoredReviewPRs }),
    autoGenerateTagTitles: () => request("POST", "/api/config/auto-generate-titles"),
    getViewer: () => request("GET", "/api/viewer"),
    getOrgs: () => request("GET", "/api/orgs"),
    getReviewRequests: () => request("GET", "/api/review-requests"),
    ignoreReviewRequest: (owner, repo, number) => request("POST", `${reviewRequestPath(owner, repo, number)}/ignore`),
    unignoreReviewRequest: (owner, repo, number) => request("POST", `${reviewRequestPath(owner, repo, number)}/unignore`),
  };
})();

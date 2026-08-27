package githubapi

const prDetailQuery = `
query PRDetail($owner: String!, $repo: String!, $number: Int!, $threadsCursor: String, $commentsCursor: String) {
  viewer { login }
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      id
      title
      url
      state
      reviewDecision
      headRefName
      body
      author { login }
      reviewThreads(first: 50, after: $threadsCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isResolved
          comments(first: 50) {
            pageInfo { hasNextPage endCursor }
            nodes { id author { login } body createdAt }
          }
        }
      }
      comments(first: 50, after: $commentsCursor) {
        pageInfo { hasNextPage endCursor }
        nodes { id author { login } body createdAt }
      }
    }
  }
}`

const threadCommentsQuery = `
query ThreadComments($threadId: ID!, $cursor: String) {
  node(id: $threadId) {
    ... on PullRequestReviewThread {
      comments(first: 50, after: $cursor) {
        pageInfo { hasNextPage endCursor }
        nodes { id author { login } body createdAt }
      }
    }
  }
}`

const viewerQuery = `
query Viewer {
  viewer { login }
}`

const viewerOrgsQuery = `
query ViewerOrgs($cursor: String) {
  viewer {
    organizations(first: 100, after: $cursor) {
      pageInfo { hasNextPage endCursor }
      nodes { login }
    }
  }
}`

const searchQuery = `
query Search($query: String!, $cursor: String) {
  search(query: $query, type: ISSUE, first: 50, after: $cursor) {
    pageInfo { hasNextPage endCursor }
    nodes {
      ... on PullRequest {
        number
        title
        url
        state
        repository { owner { login } name }
      }
    }
  }
}`

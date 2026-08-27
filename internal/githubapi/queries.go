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

const searchOpenAuthoredQuery = `
query ImportSearch($query: String!, $cursor: String) {
  search(query: $query, type: ISSUE, first: 50, after: $cursor) {
    pageInfo { hasNextPage endCursor }
    nodes {
      ... on PullRequest {
        number
        repository { owner { login } name }
      }
    }
  }
}`

# PR Checklist

A personal, local-first dashboard for tracking whether GitHub PR review threads are actually resolved. The standard GitHub PR UI sometimes hides or fails to refresh comment/thread state, making it hard to trust "is everything actually resolved?" before merging — this tool pulls PR data down and tracks resolution status independently, including a "resolved locally" state for threads you consider settled even if the original commenter hasn't formally resolved them on GitHub yet.

Single Go binary, embedded frontend, no database. All state is plain JSON files on disk so it's easy to inspect, back up, or hand-edit.

## Features

- Add a PR by pasting its URL, or use **Import** to pull in your own open PRs from a configured list of repos.
- Group tracked PRs by tag on the dashboard (a PR can have any number of tags).
- Branches named `EMBED-<number>-...` are auto-tagged with that Jira key, and any `EMBED-####`-shaped tag links out to `https://boxcast.atlassian.net/browse/<tag>`.
- Per-thread state: `unresolved` / `resolved locally` / `resolved`. Marking a thread resolved locally is purely local bookkeeping — it never calls GitHub's API. If GitHub itself reports a thread resolved, that always wins.
- If a thread you marked resolved locally gets a new reply from someone other than you, it automatically reverts to `unresolved` on the next refresh.
- Status badge (per PR, and on the dashboard) is gray (closed/merged), red (unresolved threads or changes requested), green (fully resolved and approved), or amber (anything in between).
- Manual refresh only — no background polling.
- Bulk "Remove closed" to clear out closed/merged PRs from the dashboard.

## Requirements

- Go 1.22+ (for local builds)
- A GitHub Personal Access Token with read access to the PRs/repos you want to track
- Docker (optional, for containerized runs)

## Building and running locally

```bash
go build -o pr-checklist ./cmd/server
GITHUB_TOKEN=<your-pat> DATA_DIR=./data ./pr-checklist
```

Or run directly without a separate build step:

```bash
GITHUB_TOKEN=<your-pat> DATA_DIR=./data go run ./cmd/server
```

Then open http://localhost:8080.

### Environment variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `GITHUB_TOKEN` | yes | — | GitHub PAT used for all GraphQL API calls. Server fails to start if unset. |
| `DATA_DIR` | no | `/data` | Directory where PR JSON files and `config.json` are stored. |
| `PORT` | no | `8080` | HTTP port to listen on. |

## Running in Docker

Build the image:

```bash
docker build -t pr-checklist .
```

Run it with a host directory mounted for persistent storage:

```bash
docker run -e GITHUB_TOKEN=<your-pat> -v $(pwd)/data:/data -p 8080:8080 pr-checklist
```

State (tracked PRs, tags, local resolve status, the import repo allow-list) lives entirely in the mounted `./data` directory on the host, so it survives container restarts/rebuilds.

## Configuring Import

The **Import** button on the dashboard searches a configured list of repos for your own open PRs and adds any not already tracked. Edit the allow-list from the dashboard's **Settings** panel, or by hand-editing `config.json` in your data directory:

```json
{ "importRepos": ["owner/repo", "another-owner/another-repo"] }
```

`config.json` is re-read on every Import, so hand-edits take effect immediately without restarting the server.

## Development

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

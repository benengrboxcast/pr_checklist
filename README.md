# PR Checklist

A personal, local-first dashboard for tracking whether GitHub PR review threads are actually resolved. The standard GitHub PR UI sometimes hides or fails to refresh comment/thread state, making it hard to trust "is everything actually resolved?" before merging — this tool pulls PR data down and tracks resolution status independently, including a "resolved locally" state for threads you consider settled even if the original commenter hasn't formally resolved them on GitHub yet.

Single Go binary, embedded frontend, no database. All state is plain JSON files on disk so it's easy to inspect, back up, or hand-edit.

## Features

- Add a PR by pasting its URL, or use **Import** to pull in your own open PRs from a configured list of repos and/or a checklist of GitHub orgs you belong to.
- Group tracked PRs by tag on the dashboard (a PR can have any number of tags).
- Branches starting with a Jira-style issue key (2-10 uppercase letters, a hyphen, then digits — e.g. `EMBED-1234`, `PROJ-42`) are auto-tagged with that key, and any tag of that shape links out to `https://boxcast.atlassian.net/browse/<tag>`.
- A PR's description is also scanned for a "Relevant Jira issue(s):" section, and every issue key found there is auto-tagged too — see [Description tagging](#description-tagging).
- Custom display titles per label (e.g. show "Start Stop Extend" instead of the raw `EMBED-2575` tag) — set from Settings, or auto-derived from a branch named `EMBED-2575_Rest_Of_Title` (underscores become spaces, first word capitalized). Hover any label chip to see its raw tag.
- Tag-section badges on the dashboard aggregate every PR in that section: "Needs Attention" if any is red, else "In Progress" if any is amber, else "Approved" if every PR in the section is green.
- Per-thread state: `unresolved` / `resolved locally` / `resolved`. Marking a thread resolved locally is purely local bookkeeping — it never calls GitHub's API. If GitHub itself reports a thread resolved, that always wins.
- If a thread you marked resolved locally gets a new reply from someone other than you, it automatically reverts to `unresolved` on the next refresh.
- Status badge (per PR, and on the dashboard) is gray "Closed" (closed/merged), red "Needs Attention" (unresolved threads or changes requested), green "Approved" (fully resolved and approved), or amber "In Progress" (anything in between).
- Manual refresh only — no background polling.
- Bulk "Remove closed" to clear out closed/merged PRs from the dashboard.
- Two tabs next to the "PR Checklist" title — **Opened by me** and **Review requested** — switch between the dashboard and a live list of open PRs where you've been asked to review, or that you've already reviewed; whichever is showing is highlighted. See [Review requested view](#review-requested-view).

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

The **Import** button on the dashboard searches your configured repos and orgs for your own open PRs and adds any not already tracked. Configure this from the dashboard's **Settings** panel:

- **Import from orgs** — a checklist of every GitHub org your token's user belongs to (fetched live from GitHub). Checking an org imports your open PRs from *every* repo in it. Seeing this list requires the token to have the `read:org` scope; without it (or for orgs where you only have public, non-visible membership) the checklist may come back empty or fail to load — the repo allow-list below still works either way.
- **Import repo allow-list** — individual `owner/repo` entries, for repos outside any checked org.

Both can also be hand-edited in `config.json` in your data directory:

```json
{ "importRepos": ["owner/repo", "another-owner/another-repo"], "importOrgs": ["some-org"] }
```

`config.json` is re-read on every Import, so hand-edits take effect immediately without restarting the server.

## Label titles

By default a label chip shows its raw tag text (e.g. `EMBED-2575`). You can set a custom display title per tag from Settings' **Label titles** section — every tag currently in use (or with a title already set) gets a text input; leave it blank to fall back to the raw tag. Hover any chip anywhere in the UI to see its raw tag value in a tooltip.

Titles are also derived automatically, following these rules, whenever a PR is added, refreshed, or imported:

1. The branch-name auto-tag rule applies first: if the branch starts with a Jira-style issue key (2-10 uppercase letters, a hyphen, then digits — e.g. `EMBED-1234`, `PROJ-42`), that key becomes a tag.
2. If that key is immediately followed by an underscore in the branch name (e.g. `EMBED-2575_Start_Stop_Extend`), everything after the underscore becomes that tag's title: underscores are replaced with spaces, and the first word is capitalized (the rest of the branch text's casing is left as-is — so branches already written like `EMBED-2575_Start_Stop_Extend` reproduce the exact title). A branch using a hyphen instead (`EMBED-2575-start-stop-extend`) only gets the tag, no title.

This never overwrites a title you (or a previous auto-derivation) already set. Settings also has an **Auto-generate from branch names** button that retroactively applies rule 2 above across every currently tracked PR's stored branch name — useful for PRs added before this feature existed, since it works entirely from already-stored data with no extra GitHub calls.

## Description tagging

A PR's description is scanned for a "Relevant Jira issue(s):" section (case-insensitive, colon optional) — everything from that marker up to the next blank line is searched for Jira-style issue keys (2-10 uppercase letters, a hyphen, then digits), and each one found is added as a tag. This runs automatically whenever a PR is added, refreshed, or imported, alongside the branch-name auto-tag rule above — it's purely additive and never removes a tag.

Two ways to trigger it manually, both working entirely from the description already stored locally (no GitHub call):

- **Tag from description** button on a PR's detail page — re-scans just that PR's stored description.
- **Auto-tag from descriptions** button in Settings — re-scans every currently tracked PR's stored description. Useful for PRs tracked before this feature existed.

Since both work from the locally stored description, refresh a PR first if its description changed on GitHub since it was last synced.

## Review requested view

Click the **Review requested** tab next to the "PR Checklist" title — a list of every open PR, across every repo your token can see, where you've been asked to review or have already submitted a review. Click **Opened by me** to go back to the dashboard; whichever tab matches the current view is highlighted.

This view is intentionally simple and separate from the tracked-PR dashboard:

- **Not stored locally at all.** It's a live GitHub search (`review-requested:@me` and `reviewed-by:@me`) run fresh every time you open the view or click Refresh — no JSON file, no tags, no thread tracking, no config. It also isn't scoped by the Import repo/org allow-list, since a colleague could ask you to review a PR in any repo you have access to.
- Two states only: 🟡 **Review Requested** (you currently have a pending review request) and 🟢 **Reviewed** (you've already submitted a review and there's no new pending request). If you reviewed a PR before and were re-requested after new commits, it shows as Review Requested again — that's the current actionable state.
- Each entry links straight to the PR on GitHub rather than into this app's detail view, since these PRs aren't tracked records here.

## Development

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .
```

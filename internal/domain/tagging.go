package domain

import "regexp"

var (
	embedBranchPattern = regexp.MustCompile(`^EMBED-(\d+)`)
	embedTagPattern    = regexp.MustCompile(`^EMBED-\d+$`)
)

// ExtractEmbedTag returns the "EMBED-<digits>" tag implied by a branch name,
// and whether the branch name matched at all. A branch like
// "EMBED-1234-fix-thing" yields ("EMBED-1234", true).
func ExtractEmbedTag(branchName string) (string, bool) {
	m := embedBranchPattern.FindStringSubmatch(branchName)
	if m == nil {
		return "", false
	}
	return "EMBED-" + m[1], true
}

// IsEmbedTag reports whether a tag name is exactly of the EMBED-<digits> shape,
// used both for auto-tagging and for deciding whether to render a Jira link.
func IsEmbedTag(tag string) bool {
	return embedTagPattern.MatchString(tag)
}

// EnsureTag returns tags with want appended if not already present, leaving
// the input order/contents otherwise untouched (additive, idempotent).
func EnsureTag(tags []string, want string) []string {
	for _, t := range tags {
		if t == want {
			return tags
		}
	}
	return append(tags, want)
}

// RemoveTag returns tags with target removed, if present.
func RemoveTag(tags []string, target string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t != target {
			out = append(out, t)
		}
	}
	return out
}

package domain

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// jiraBranchPattern matches a Jira-style issue key at the very start of a
	// branch name: 2-10 uppercase letters, a hyphen, then digits.
	jiraBranchPattern = regexp.MustCompile(`^([A-Z]{2,10})-(\d+)`)
	jiraTagPattern    = regexp.MustCompile(`^[A-Z]{2,10}-\d+$`)
	jiraTitlePattern  = regexp.MustCompile(`^[A-Z]{2,10}-\d+_(.+)$`)
	// jiraAnywherePattern matches a Jira-style issue key anywhere in free text
	// (e.g. a PR description). The \b boundaries stop it from matching a
	// substring of a longer all-caps run, e.g. never pulling "ERLONGWORD-123"
	// out of "SUPERLONGWORD-123".
	jiraAnywherePattern = regexp.MustCompile(`\b[A-Z]{2,10}-\d+\b`)
	// jiraDescriptionSectionPattern locates the "Relevant Jira issue(s):"
	// marker (case-insensitive, colon optional) that introduces the section
	// of a PR description listing its related issues.
	jiraDescriptionSectionPattern = regexp.MustCompile(`(?i)relevant jira issue\(s\):?`)
)

// ExtractJiraTag returns the Jira-style issue key implied by a branch name
// (2-10 uppercase letters, a hyphen, then digits), and whether the branch
// name matched at all. A branch like "EMBED-1234-fix-thing" yields
// ("EMBED-1234", true).
func ExtractJiraTag(branchName string) (string, bool) {
	m := jiraBranchPattern.FindStringSubmatch(branchName)
	if m == nil {
		return "", false
	}
	return m[1] + "-" + m[2], true
}

// ExtractJiraTitle derives a human-readable display title from the part of a
// branch name after "<ISSUE-KEY>_", if the branch uses that
// underscore-separated convention. Underscores become spaces and only the
// first word is forced to start with an uppercase letter — the rest of the
// branch name's casing is preserved as-is. A branch like
// "EMBED-2575_Start_Stop_Extend" yields ("Start Stop Extend", true); a branch
// like "EMBED-1234-fix-thing" (hyphen, not underscore) yields ("", false).
func ExtractJiraTitle(branchName string) (string, bool) {
	m := jiraTitlePattern.FindStringSubmatch(branchName)
	if m == nil {
		return "", false
	}
	title := strings.TrimSpace(strings.ReplaceAll(m[1], "_", " "))
	if title == "" {
		return "", false
	}
	return capitalizeFirst(title), true
}

func capitalizeFirst(s string) string {
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// IsJiraTag reports whether a tag name is exactly of the Jira issue key shape
// (2-10 uppercase letters, a hyphen, then digits), used both for auto-tagging
// and for deciding whether to render a Jira link.
func IsJiraTag(tag string) bool {
	return jiraTagPattern.MatchString(tag)
}

// ExtractJiraTagsFromDescription finds every Jira-style issue key (e.g.
// "EMBED-1234") mentioned in a PR description's "Relevant Jira issue(s):"
// section, if that section exists. The section runs from the marker to the
// next blank line (or the end of the description if it's the last section).
// Returns nil if the marker isn't present. Duplicate keys are removed,
// preserving first-seen order.
func ExtractJiraTagsFromDescription(description string) []string {
	loc := jiraDescriptionSectionPattern.FindStringIndex(description)
	if loc == nil {
		return nil
	}
	section := description[loc[1]:]
	if idx := strings.Index(section, "\n\n"); idx != -1 {
		section = section[:idx]
	}

	matches := jiraAnywherePattern.FindAllString(section, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
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

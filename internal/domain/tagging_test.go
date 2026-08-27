package domain

import (
	"reflect"
	"testing"
)

func TestExtractJiraTag(t *testing.T) {
	tests := []struct {
		branch string
		want   string
		wantOk bool
	}{
		{"EMBED-1234-fix-thing", "EMBED-1234", true},
		{"EMBED-12345-foo", "EMBED-12345", true},
		{"EMBED-7", "EMBED-7", true},
		{"AB-7", "AB-7", true},                 // minimum-length (2 char) prefix
		{"ABCDEFGHIJ-7", "ABCDEFGHIJ-7", true}, // maximum-length (10 char) prefix
		{"EMBEDX-1234", "EMBEDX-1234", true},   // any 2-10 uppercase letters qualify now, not just "EMBED"
		{"PROJ-42-do-a-thing", "PROJ-42", true},
		{"A-1234", "", false},           // single-letter prefix, too short
		{"ABCDEFGHIJK-1234", "", false}, // 11-letter prefix, too long
		{"embed-1234-lowercase", "", false},
		{"feature/EMBED-1234", "", false}, // must anchor at start of branch name
		{"main", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.branch, func(t *testing.T) {
			got, ok := ExtractJiraTag(tc.branch)
			if got != tc.want || ok != tc.wantOk {
				t.Errorf("ExtractJiraTag(%q) = (%q, %v), want (%q, %v)", tc.branch, got, ok, tc.want, tc.wantOk)
			}
		})
	}
}

func TestIsJiraTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"EMBED-1234", true},
		{"EMBED-1", true},
		{"AB-1", true},
		{"ABCDEFGHIJ-1", true},
		{"ABCDEFGHIJK-1", false}, // 11-letter prefix, too long
		{"A-1", false},           // single-letter prefix, too short
		{"backend", false},
		{"EMBED-1234-extra", false},
		{"embed-1234", false}, // lowercase
		{"EMBED-", false},
	}
	for _, tc := range tests {
		if got := IsJiraTag(tc.tag); got != tc.want {
			t.Errorf("IsJiraTag(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}

func TestExtractJiraTitle(t *testing.T) {
	tests := []struct {
		branch string
		want   string
		wantOk bool
	}{
		{"EMBED-2575_Start_Stop_Extend", "Start Stop Extend", true},
		{"EMBED-2575_start_stop_extend", "Start stop extend", true}, // only first word forced uppercase
		{"EMBED-7_fix", "Fix", true},
		{"PROJ-42_Some_Title", "Some Title", true},
		{"EMBED-1234-fix-thing", "", false}, // hyphen, not underscore right after the digits
		{"EMBED-1234", "", false},           // nothing after the digits at all
		{"EMBED-1234_", "", false},          // underscore but nothing after it
		{"EMBED-1234___", "", false},        // only underscores after
		{"embed-1234_lowercase_tag", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.branch, func(t *testing.T) {
			got, ok := ExtractJiraTitle(tc.branch)
			if got != tc.want || ok != tc.wantOk {
				t.Errorf("ExtractJiraTitle(%q) = (%q, %v), want (%q, %v)", tc.branch, got, ok, tc.want, tc.wantOk)
			}
		})
	}
}

func TestExtractJiraTagsFromDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        []string
	}{
		{
			name:        "bulleted list",
			description: "Some intro text.\n\nRelevant Jira issue(s):\n- EMBED-1234\n- PROJ-42\n\nMore stuff after.",
			want:        []string{"EMBED-1234", "PROJ-42"},
		},
		{
			name:        "inline comma list",
			description: "Relevant Jira issue(s): EMBED-1234, PROJ-42",
			want:        []string{"EMBED-1234", "PROJ-42"},
		},
		{
			name:        "case-insensitive marker, section is last in description",
			description: "Body text.\n\nRELEVANT JIRA ISSUE(S):\nEMBED-9999",
			want:        []string{"EMBED-9999"},
		},
		{
			name:        "marker with no colon",
			description: "Relevant Jira issue(s)\nEMBED-1",
			want:        []string{"EMBED-1"},
		},
		{
			name:        "duplicate issue keys deduped, order preserved",
			description: "Relevant Jira issue(s):\n- EMBED-1234\n- EMBED-1234\n- PROJ-42",
			want:        []string{"EMBED-1234", "PROJ-42"},
		},
		{
			name:        "no marker present",
			description: "Just a normal PR description mentioning EMBED-1234 in passing.",
			want:        nil,
		},
		{
			name:        "marker present but nothing matches inside its section",
			description: "Relevant Jira issue(s):\nnone this time\n\nEMBED-1234 mentioned later, outside the section",
			want:        nil,
		},
		{
			name:        "empty description",
			description: "",
			want:        nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractJiraTagsFromDescription(tc.description)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ExtractJiraTagsFromDescription(%q) = %v, want %v", tc.description, got, tc.want)
			}
		})
	}
}

func TestEnsureTagIsAdditiveAndIdempotent(t *testing.T) {
	tags := []string{"backend"}
	tags = EnsureTag(tags, "EMBED-1234")
	if !reflect.DeepEqual(tags, []string{"backend", "EMBED-1234"}) {
		t.Fatalf("unexpected tags after first EnsureTag: %v", tags)
	}
	tags = EnsureTag(tags, "EMBED-1234")
	if !reflect.DeepEqual(tags, []string{"backend", "EMBED-1234"}) {
		t.Fatalf("EnsureTag was not idempotent: %v", tags)
	}
}

func TestRemoveTag(t *testing.T) {
	tags := []string{"backend", "EMBED-1234"}
	tags = RemoveTag(tags, "EMBED-1234")
	if !reflect.DeepEqual(tags, []string{"backend"}) {
		t.Fatalf("unexpected tags after RemoveTag: %v", tags)
	}
}

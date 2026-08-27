package domain

import (
	"reflect"
	"testing"
)

func TestExtractEmbedTag(t *testing.T) {
	tests := []struct {
		branch string
		want   string
		wantOk bool
	}{
		{"EMBED-1234-fix-thing", "EMBED-1234", true},
		{"EMBED-12345-foo", "EMBED-12345", true},
		{"EMBED-7", "EMBED-7", true},
		{"embed-1234-lowercase", "", false},
		{"feature/EMBED-1234", "", false}, // must anchor at start of branch name
		{"EMBEDX-1234", "", false},
		{"main", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.branch, func(t *testing.T) {
			got, ok := ExtractEmbedTag(tc.branch)
			if got != tc.want || ok != tc.wantOk {
				t.Errorf("ExtractEmbedTag(%q) = (%q, %v), want (%q, %v)", tc.branch, got, ok, tc.want, tc.wantOk)
			}
		})
	}
}

func TestIsEmbedTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"EMBED-1234", true},
		{"EMBED-1", true},
		{"backend", false},
		{"EMBED-1234-extra", false},
		{"EMBED-", false},
	}
	for _, tc := range tests {
		if got := IsEmbedTag(tc.tag); got != tc.want {
			t.Errorf("IsEmbedTag(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}

func TestExtractEmbedTitle(t *testing.T) {
	tests := []struct {
		branch string
		want   string
		wantOk bool
	}{
		{"EMBED-2575_Start_Stop_Extend", "Start Stop Extend", true},
		{"EMBED-2575_start_stop_extend", "Start stop extend", true}, // only first word forced uppercase
		{"EMBED-7_fix", "Fix", true},
		{"EMBED-1234-fix-thing", "", false}, // hyphen, not underscore right after the digits
		{"EMBED-1234", "", false},           // nothing after the digits at all
		{"EMBED-1234_", "", false},          // underscore but nothing after it
		{"EMBED-1234___", "", false},        // only underscores after
		{"embed-1234_lowercase_tag", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.branch, func(t *testing.T) {
			got, ok := ExtractEmbedTitle(tc.branch)
			if got != tc.want || ok != tc.wantOk {
				t.Errorf("ExtractEmbedTitle(%q) = (%q, %v), want (%q, %v)", tc.branch, got, ok, tc.want, tc.wantOk)
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

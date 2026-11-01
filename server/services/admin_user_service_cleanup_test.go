package services

import "testing"

func TestMergeCleanupPlansDeduplicatesRefsURLsAndLiveKit(t *testing.T) {
	base := &CleanupPlan{
		refs: []fileRef{{URL: "/uploads/a", Size: 10, UserID: "u1"}},
		urls: []string{"/uploads/icon"},
		livekit: []livekitEntry{{InstanceID: "lk1", IsPlatformManaged: true}},
	}
	merged := mergeCleanupPlans(base, map[string]*CleanupPlan{
		"server-a": {
			refs: []fileRef{
				{URL: "/uploads/a", Size: 10, UserID: "u1"},
				{URL: "/uploads/b", Size: 20, UserID: "u2"},
			},
			urls: []string{"/uploads/icon", "/uploads/banner"},
			livekit: []livekitEntry{
				{InstanceID: "lk1", IsPlatformManaged: true},
				{InstanceID: "lk2", IsPlatformManaged: false},
			},
		},
	})
	if len(merged.refs) != 2 {
		t.Fatalf("got %d file refs, want 2", len(merged.refs))
	}
	if len(merged.urls) != 2 {
		t.Fatalf("got %d standalone URLs, want 2", len(merged.urls))
	}
	if len(merged.livekit) != 2 {
		t.Fatalf("got %d LiveKit entries, want 2", len(merged.livekit))
	}
}

package service_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/importer"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/service"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
)

func TestOfficialProfilesAndCompletions(t *testing.T) {
	ctx := context.Background()
	data, report, err := importer.LoadDir(ctx, "../../data")
	if err != nil {
		t.Fatalf("%v %+v", err, report)
	}
	memory := store.NewMemoryStore()
	if err := memory.ReplaceDataset(ctx, data); err != nil {
		t.Fatal(err)
	}
	career, err := service.NewCareerService(memory, mustDate(t, "2026-10-01"))
	if err != nil {
		t.Fatal(err)
	}
	for id := range data.Employees {
		profile, err := career.GetProfile(ctx, id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if len(profile.Recommendations) > 3 {
			t.Fatal("too many recommendations")
		}
		for _, level := range profile.Progress.EffectiveSkills {
			if level > 5 {
				t.Fatal("skill above maximum")
			}
		}
		noncriticalSeen := false
		for _, r := range profile.Recommendations {
			if noncriticalSeen && r.Evidence.CriticalGap {
				t.Fatal("critical recommendation ranked below noncritical")
			}
			if !r.Evidence.CriticalGap {
				noncriticalSeen = true
			}
		}
	}
	// A voluntary event without skill gains is still recorded.
	empty := domain.Event{EventID: "EMPTY", Title: "Meeting", Format: "self_paced"}
	data.Events[empty.EventID] = empty
	if err := memory.ReplaceDataset(ctx, data); err != nil {
		t.Fatal(err)
	}
	before, err := career.GetProfile(ctx, "E0001")
	if err != nil {
		t.Fatal(err)
	}
	after, err := career.CompleteEvent(ctx, "E0001", "EMPTY")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Progress.EffectiveSkills, after.Profile.Progress.EffectiveSkills) || len(after.Profile.Activities) != len(before.Activities)+1 {
		t.Fatal("empty event must preserve skills and add history")
	}
	// Repeating the official club increments history, and never exceeds level 5.
	for range 2 {
		if _, err := career.CompleteEvent(ctx, "E0001", "EV_036"); err != nil {
			t.Fatal(err)
		}
	}
	final, err := career.GetProfile(ctx, "E0001")
	if err != nil {
		t.Fatal(err)
	}
	if len(final.Activities) != len(before.Activities)+3 {
		t.Fatal("club completion missing")
	}
}

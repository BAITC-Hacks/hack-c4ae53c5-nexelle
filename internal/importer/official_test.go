package importer_test

import (
	"context"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/importer"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialDatasetLoadsExactSchema(t *testing.T) {
	data, report, err := importer.LoadDir(context.Background(), "../../data")
	if err != nil {
		t.Fatalf("%v: %+v", err, report.Errors)
	}
	stats := data.Stats()
	if stats.Employees != 200 || stats.Events != 40 || stats.Skills != 60 || stats.RoleProfiles != 32 || stats.ActivityRecords != 2743 {
		t.Fatalf("unexpected official dataset: %+v", stats)
	}
	optional, mandatory, prereqs, paced, gains := 0, 0, 0, 0, 0
	for _, e := range data.Events {
		if e.Mandatory {
			mandatory++
		} else {
			optional++
		}
		if len(e.Prerequisites) > 0 {
			prereqs++
		}
		if e.Format == "self_paced" {
			paced++
		}
		gains += len(e.DevelopsSkills)
		if len(e.TargetRoles) == 0 || len(e.TargetGrades) == 0 {
			t.Fatal("audience lost")
		}
	}
	if optional != 36 || mandatory != 4 || prereqs != 12 || paced != 9 || gains == 0 {
		t.Fatalf("counts %d %d %d %d %d", optional, mandatory, prereqs, paced, gains)
	}
}

func TestRejectsLegacyEventKeysRatherThanSilentlyIgnoring(t *testing.T) {
	for _, field := range []string{"target_roles", "target_grades", "develops_skills"} {
		t.Run(field, func(t *testing.T) {
			dir := writeDataset(t, "")
			path := filepath.Join(dir, "events.json")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			content = []byte(strings.ReplaceAll(string(content), field, "legacy_"+field))
			if err := os.WriteFile(path, content, 0600); err != nil {
				t.Fatal(err)
			}
			_, _, err = importer.LoadDir(context.Background(), dir)
			if err == nil {
				t.Fatal("missing required canonical field was accepted")
			}
		})
	}
}

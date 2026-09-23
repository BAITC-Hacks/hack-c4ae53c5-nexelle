package config

import (
	"testing"
	"time"
)

func TestLoadUsesFixedDatasetSnapshotDate(t *testing.T) {
	t.Setenv("SNAPSHOT_DATE", "")
	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if actual := config.SnapshotDate.Format(time.DateOnly); actual != defaultSnapshotDate {
		t.Fatalf("SnapshotDate = %s, want %s", actual, defaultSnapshotDate)
	}
}

func TestLoadRejectsDifferentSnapshotDate(t *testing.T) {
	t.Setenv("SNAPSHOT_DATE", "2026-10-02")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want fixed snapshot date validation error")
	}
}

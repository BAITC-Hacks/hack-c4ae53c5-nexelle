package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultSnapshotDate = "2026-10-01"

type Config struct {
	Port              int
	DataDir           string
	SnapshotDate      time.Time
	ReadHeaderTimeout time.Duration
}

func Load() (Config, error) {
	port, err := intFromEnv("APP_PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	snapshotDate, err := fixedSnapshotDate()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:              port,
		DataDir:           stringFromEnv("DATA_DIR", "./data"),
		SnapshotDate:      snapshotDate,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

func (c Config) Address() string {
	return net.JoinHostPort("", strconv.Itoa(c.Port))
}

func intFromEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	result, err := strconv.Atoi(value)
	if err != nil || result < 1 || result > 65535 {
		return 0, fmt.Errorf("%s must be a valid TCP port", name)
	}
	return result, nil
}

func fixedSnapshotDate() (time.Time, error) {
	if configured := strings.TrimSpace(os.Getenv("SNAPSHOT_DATE")); configured != "" && configured != defaultSnapshotDate {
		return time.Time{}, fmt.Errorf("SNAPSHOT_DATE must be %s for the Career Quest dataset", defaultSnapshotDate)
	}
	result, err := time.Parse(time.DateOnly, defaultSnapshotDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse fixed snapshot date: %w", err)
	}
	return result, nil
}

func stringFromEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

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

	snapshotDate, err := dateFromEnv("SNAPSHOT_DATE", defaultSnapshotDate)
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

func dateFromEnv(name, fallback string) (time.Time, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		value = fallback
	}

	result, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must have format YYYY-MM-DD: %w", name, err)
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

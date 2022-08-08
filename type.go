package main

import "time"

// Event represents a single notification.
type Event struct {
	Name string // Relative path to the file or directory.
	Op   string // File operation that triggered the event.
}

type redresser struct {
	mountPath    string
	originalPath string
	checksum     string
}

type watchList struct {
	MountPath    string
	OriginalPath string
}

type redresserConfig struct {
	Enabled         bool
	PollingInterval time.Duration `json:"polling_interval"`
	WatchList       []watchList   `json:"watchlist"`
}

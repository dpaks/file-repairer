package main

// Event represents a single notification.
type Event struct {
	Name string // Relative path to the file or directory.
	Op   string // File operation that triggered the event.
}

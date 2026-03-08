package model

import "time"

// ProcessStat holds a single sample of process metrics
type ProcessStat struct {
	Timestamp  time.Time
	PID        int32
	Name       string
	CPUPercent float64 // percentage 0-100
	MemRSS     uint64  // Resident Set Size in bytes
	MemVMS     uint64  // Virtual Memory Size in bytes
	MemPercent float32 // memory percentage
	Status     string  // running, sleeping, etc.
}

// Config holds the application configuration
type Config struct {
	ProcessNames []string // process names to monitor
	Duration     int      // monitoring duration in seconds (0 = unlimited)
	OutputFile   string   // output Excel file path
	Interval     int      // sampling interval in seconds (default 1)
}

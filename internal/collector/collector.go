package collector

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"ProcessMonitor/internal/model"

	"github.com/shirou/gopsutil/v3/process"
)

// Collector monitors specified processes and collects stats
type Collector struct {
	config   *model.Config
	mu       sync.Mutex
	allStats []model.ProcessStat
	stopCh   chan struct{}
	prevCPU  map[int32]float64 // track previous CPU for delta calculation
}

// New creates a new Collector
func New(cfg *model.Config) *Collector {
	return &Collector{
		config:  cfg,
		stopCh:  make(chan struct{}),
		prevCPU: make(map[int32]float64),
	}
}

// Start begins collecting process stats at the configured interval
func (c *Collector) Start() {
	ticker := time.NewTicker(time.Duration(c.config.Interval) * time.Second)
	defer ticker.Stop()

	var durationTimer <-chan time.Time
	if c.config.Duration > 0 {
		durationTimer = time.After(time.Duration(c.config.Duration) * time.Second)
	}

	fmt.Printf("Started monitoring: %s\n", strings.Join(c.config.ProcessNames, ", "))
	fmt.Println("Press Ctrl+C to stop and save results...")
	fmt.Println(strings.Repeat("-", 70))

	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-durationTimer:
			fmt.Println("\nMonitoring duration reached.")
			return
		case <-c.stopCh:
			return
		}
	}
}

// Stop signals the collector to stop
func (c *Collector) Stop() {
	close(c.stopCh)
}

// GetStats returns a copy of all collected stats
func (c *Collector) GetStats() []model.ProcessStat {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]model.ProcessStat, len(c.allStats))
	copy(result, c.allStats)
	return result
}

// collect gathers one sample for all matching processes
func (c *Collector) collect() {
	now := time.Now()
	procs, err := process.Processes()
	if err != nil {
		fmt.Printf("Error listing processes: %v\n", err)
		return
	}

	found := make(map[string]bool)
	for _, name := range c.config.ProcessNames {
		found[strings.ToLower(name)] = false
	}

	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}

		// Check if this process matches any monitored name
		nameLower := strings.ToLower(name)
		matched := false
		for _, target := range c.config.ProcessNames {
			if strings.ToLower(target) == nameLower ||
				strings.Contains(nameLower, strings.ToLower(target)) {
				matched = true
				found[strings.ToLower(target)] = true
				break
			}
		}
		if !matched {
			continue
		}

		stat, err := c.buildStat(p, name, now)
		if err != nil {
			continue
		}

		c.mu.Lock()
		c.allStats = append(c.allStats, stat)
		c.mu.Unlock()

		fmt.Printf("[%s] PID:%-7d %-20s CPU:%6.2f%%  RSS:%8s  MEM:%.2f%%\n",
			now.Format("15:04:05"),
			stat.PID,
			stat.Name,
			stat.CPUPercent,
			formatBytes(stat.MemRSS),
			stat.MemPercent,
		)
	}

	// Warn about processes not found
	for name, wasFound := range found {
		if !wasFound {
			fmt.Printf("[%s] Process not found: %s\n", now.Format("15:04:05"), name)
		}
	}
}

// buildStat creates a ProcessStat from a gopsutil process
func (c *Collector) buildStat(p *process.Process, name string, ts time.Time) (model.ProcessStat, error) {
	// CPU percent with 1s interval for accurate measurement
	cpuPct, err := p.CPUPercent()
	if err != nil {
		cpuPct = 0
	}

	memInfo, err := p.MemoryInfo()
	if err != nil {
		return model.ProcessStat{}, fmt.Errorf("memory info: %w", err)
	}

	memPct, err := p.MemoryPercent()
	if err != nil {
		memPct = 0
	}

	statusSlice, err := p.Status()
	var statusStr string
	if err == nil && len(statusSlice) > 0 && statusSlice[0] != "" {
		statusStr = statusSlice[0]
	} else {
		// Fallback to IsRunning() when Status() is unavailable
		running, rerr := p.IsRunning()
		if rerr == nil {
			if running {
				statusStr = "running"
			} else {
				statusStr = "stopped"
			}
		} else {
			statusStr = "unknown"
		}
	}

	return model.ProcessStat{
		Timestamp:  ts,
		PID:        p.Pid,
		Name:       name,
		CPUPercent: cpuPct,
		MemRSS:     memInfo.RSS,
		MemVMS:     memInfo.VMS,
		MemPercent: memPct,
		Status:     statusStr,
	}, nil
}

// formatBytes converts bytes to megabytes string (M)
func formatBytes(b uint64) string {
	mb := float64(b) / (1024.0 * 1024.0)
	return fmt.Sprintf("%.2f M", mb)
}

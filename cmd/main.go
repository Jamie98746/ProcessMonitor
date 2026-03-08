package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ProcessMonitor/internal/collector"
	"ProcessMonitor/internal/exporter"
	"ProcessMonitor/internal/model"
)

const banner = `
╔══════════════════════════════════════════════╗
║         Process Monitor — procmon            ║
║  CPU & Memory tracker → Excel Report         ║
╚══════════════════════════════════════════════╝
`

func main() {
	fmt.Print(banner)

	// --- CLI flags ---
	processFlag := flag.String("p", "", "Comma-separated process names to monitor (e.g. chrome,notepad)")
	durationFlag := flag.Int("d", 0, "Monitoring duration in seconds (0 = run until Ctrl+C)")
	outputFlag := flag.String("o", "", "Output Excel file path (default: procmon_<timestamp>.xlsx)")
	intervalFlag := flag.Int("i", 1, "Sampling interval in seconds (default: 1)")
	flag.Parse()

	if *processFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: -p flag is required")
		printUsage()
		os.Exit(1)
	}

	// Parse process names
	names := []string{}
	for _, n := range strings.Split(*processFlag, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			names = append(names, n)
		}
	}

	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid process names provided")
		os.Exit(1)
	}

	// Default output filename
	outputFile := *outputFlag
	if outputFile == "" {
		outputFile = fmt.Sprintf("procmon_%s.xlsx", time.Now().Format("20060102_150405"))
	}

	cfg := &model.Config{
		ProcessNames: names,
		Duration:     *durationFlag,
		OutputFile:   outputFile,
		Interval:     *intervalFlag,
	}

	fmt.Printf("  Processes  : %s\n", strings.Join(names, ", "))
	fmt.Printf("  Interval   : %ds\n", cfg.Interval)
	if cfg.Duration > 0 {
		fmt.Printf("  Duration   : %ds\n", cfg.Duration)
	} else {
		fmt.Printf("  Duration   : unlimited (Ctrl+C to stop)\n")
	}
	fmt.Printf("  Output     : %s\n\n", outputFile)

	c := collector.New(cfg)

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	doneCh := make(chan struct{})

	go func() {
		c.Start()
		close(doneCh)
	}()

	select {
	case <-sigCh:
		fmt.Println("\n\nInterrupted. Saving data...")
		c.Stop()
		<-doneCh
	case <-doneCh:
		fmt.Println("Saving data...")
	}

	// Export to Excel
	stats := c.GetStats()
	if len(stats) == 0 {
		fmt.Println("No data collected. Exiting without creating file.")
		os.Exit(0)
	}

	fmt.Printf("Writing %d records to %s ...\n", len(stats), outputFile)
	if err := exporter.WriteExcel(stats, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write Excel: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done! Report saved to: %s\n", outputFile)
}

func printUsage() {
	fmt.Println(`Usage:
  procmon -p <process_names> [options]

Options:
  -p string   Comma-separated process names (required)
              e.g. -p "chrome,notepad" or -p "python3"
  -d int      Duration in seconds (default 0 = unlimited, stop with Ctrl+C)
  -i int      Sampling interval in seconds (default 1)
  -o string   Output Excel file path (default auto-generated)

Examples:
  procmon -p chrome -d 60
  procmon -p "python3,nginx" -d 120 -o report.xlsx
  procmon -p notepad.exe -i 2 -d 30`)
}

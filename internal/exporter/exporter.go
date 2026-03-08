package exporter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ProcessMonitor/internal/model"

	"github.com/xuri/excelize/v2"
)

// styleIDs holds reusable style IDs
type styleIDs struct {
	header    int
	timestamp int
	number    int
	percent   int
	bytes     int
	altRow    int
	title     int
	summary   int
	summaryHd int
	good      int
	warn      int
}

// WriteExcel exports all collected stats to an Excel file
func WriteExcel(stats []model.ProcessStat, cfg *model.Config) error {
	if len(stats) == 0 {
		return fmt.Errorf("no data collected")
	}

	f := excelize.NewFile()
	defer f.Close()

	ids, err := createStyles(f)
	if err != nil {
		return fmt.Errorf("create styles: %w", err)
	}

	// Group stats by process name
	grouped := groupByProcess(stats)

	// Create summary sheet first
	if err := writeSummarySheet(f, grouped, ids); err != nil {
		return err
	}

	// Create a sheet per process
	for procName, procStats := range grouped {
		sheetName := sanitizeSheetName(procName)
		if err := writeProcessSheet(f, sheetName, procStats, ids); err != nil {
			return err
		}
	}

	// Remove default Sheet1 if we created other sheets
	sheets := f.GetSheetList()
	if len(sheets) > 1 {
		for _, s := range sheets {
			if s == "Sheet1" {
				_ = f.DeleteSheet("Sheet1")
				break
			}
		}
	}

	// Set Summary as active
	sheetIndex, _ := f.GetSheetIndex("Summary")
	if sheetIndex >= 0 {
		f.SetActiveSheet(sheetIndex)
	}

	if err := f.SaveAs(cfg.OutputFile); err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	return nil
}

func createStyles(f *excelize.File) (*styleIDs, error) {
	ids := &styleIDs{}

	// Header style - dark blue background, white bold text
	s, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10, Family: "Arial"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}
	ids.header = s

	// Timestamp style (explicitly include seconds)
	s, err = f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 9, Family: "Arial"},
		CustomNumFmt: strPtr("yyyy-mm-dd hh:mm:ss"),
		Alignment:    &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return nil, err
	}
	ids.timestamp = s

	// Number style (2 decimal places)
	s, err = f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 9, Family: "Arial"},
		NumFmt: 2,
	})
	if err != nil {
		return nil, err
	}
	ids.number = s

	// Percent style
	s, err = f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 9, Family: "Arial"},
		CustomNumFmt: strPtr("0.00\"%\""),
		Alignment:    &excelize.Alignment{Horizontal: "right"},
	})
	if err != nil {
		return nil, err
	}
	ids.percent = s

	// Bytes style (integer with comma)
	s, err = f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Size: 9, Family: "Arial"},
		NumFmt: 3, // #,##0
	})
	if err != nil {
		return nil, err
	}
	ids.bytes = s

	// Alternating row style
	s, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 9, Family: "Arial"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"EBF3FB"}, Pattern: 1},
	})
	if err != nil {
		return nil, err
	}
	ids.altRow = s

	// Title style
	s, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Family: "Arial", Color: "1F4E79"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	ids.title = s

	// Summary value style
	s, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Arial"},
		NumFmt:    2,
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	if err != nil {
		return nil, err
	}
	ids.summary = s

	// Summary header style - lighter
	s, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Family: "Arial", Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2E75B6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	ids.summaryHd = s

	// Good (green) highlight
	s, err = f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true, Size: 10, Family: "Arial"},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"E2EFDA"}, Pattern: 1},
		NumFmt: 2,
	})
	if err != nil {
		return nil, err
	}
	ids.good = s

	// Warning (orange) highlight
	s, err = f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true, Size: 10, Family: "Arial"},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"FCE4D6"}, Pattern: 1},
		NumFmt: 2,
	})
	if err != nil {
		return nil, err
	}
	ids.warn = s

	return ids, nil
}

func writeSummarySheet(f *excelize.File, grouped map[string][]model.ProcessStat, ids *styleIDs) error {
	sheet := "Summary"
	f.NewSheet(sheet)

	// Title
	f.MergeCell(sheet, "A1", "H1")
	f.SetCellValue(sheet, "A1", "🖥️  Process Monitor — Summary Report")
	f.SetCellStyle(sheet, "A1", "A1", ids.title)
	f.SetRowHeight(sheet, 1, 30)

	// Meta info
	f.SetCellValue(sheet, "A2", "Generated:")
	f.SetCellValue(sheet, "B2", time.Now())
	f.SetCellStyle(sheet, "B2", "B2", ids.timestamp)
	f.SetCellValue(sheet, "D2", "Processes Monitored:")
	f.SetCellValue(sheet, "E2", len(grouped))

	// Headers row 4
	headers := []string{"Process", "PID", "Samples", "Avg CPU %", "Max CPU %", "Avg RSS (M)", "Max RSS (M)", "Avg MEM %"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 4)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, ids.summaryHd)
	}
	f.SetRowHeight(sheet, 4, 22)

	// Sort processes for consistent output
	names := make([]string, 0, len(grouped))
	for n := range grouped {
		names = append(names, n)
	}
	sort.Strings(names)

	row := 5
	for _, name := range names {
		procStats := grouped[name]
		summ := computeSummary(procStats)

		f.SetCellValue(sheet, cellName(1, row), name)
		f.SetCellValue(sheet, cellName(2, row), summ.pid)
		f.SetCellValue(sheet, cellName(3, row), len(procStats))
		f.SetCellValue(sheet, cellName(4, row), summ.avgCPU)
		f.SetCellValue(sheet, cellName(5, row), summ.maxCPU)
		// convert bytes -> MB
		f.SetCellValue(sheet, cellName(6, row), float64(summ.avgRSS)/(1024.0*1024.0))
		f.SetCellValue(sheet, cellName(7, row), float64(summ.maxRSS)/(1024.0*1024.0))
		f.SetCellValue(sheet, cellName(8, row), summ.avgMem)

		// Highlight high CPU
		cpuStyle := ids.summary
		if summ.avgCPU > 80 {
			cpuStyle = ids.warn
		} else if summ.avgCPU < 20 {
			cpuStyle = ids.good
		}
		f.SetCellStyle(sheet, cellName(4, row), cellName(5, row), cpuStyle)
		f.SetCellStyle(sheet, cellName(6, row), cellName(7, row), ids.number)

		// Add hyperlink to process sheet
		sheetRef := sanitizeSheetName(name)
		f.SetCellFormula(sheet, cellName(1, row), fmt.Sprintf(`HYPERLINK("#'%s'!A1","%s")`, sheetRef, name))

		row++
	}

	// Column widths
	colWidths := map[string]float64{
		"A": 22, "B": 20, "C": 10, "D": 18, "E": 10, "F": 14, "G": 14, "H": 12,
	}
	for col, w := range colWidths {
		f.SetColWidth(sheet, col, col, w)
	}

	return nil
}

func writeProcessSheet(f *excelize.File, sheetName string, stats []model.ProcessStat, ids *styleIDs) error {
	f.NewSheet(sheetName)

	// Title
	f.MergeCell(sheetName, "A1", "H1")
	f.SetCellValue(sheetName, "A1", fmt.Sprintf("Process: %s  (PID: %d)", stats[0].Name, stats[0].PID))
	f.SetCellStyle(sheetName, "A1", "A1", ids.title)
	f.SetRowHeight(sheetName, 1, 28)

	// Headers
	headers := []string{"Timestamp", "PID", "Process Name", "CPU %", "RSS (M)", "VMS (M)", "Memory %", "Status"}
	for i, h := range headers {
		cell := cellName(i+1, 2)
		f.SetCellValue(sheetName, cell, h)
		f.SetCellStyle(sheetName, cell, cell, ids.header)
	}
	f.SetRowHeight(sheetName, 2, 20)

	// Data rows
	for i, stat := range stats {
		row := i + 3
		isAlt := i%2 == 1

		f.SetCellValue(sheetName, cellName(1, row), stat.Timestamp)
		f.SetCellStyle(sheetName, cellName(1, row), cellName(1, row), ids.timestamp)

		f.SetCellValue(sheetName, cellName(2, row), stat.PID)
		f.SetCellValue(sheetName, cellName(3, row), stat.Name)
		f.SetCellValue(sheetName, cellName(4, row), stat.CPUPercent)
		// convert bytes -> MB
		f.SetCellValue(sheetName, cellName(5, row), float64(stat.MemRSS)/(1024.0*1024.0))
		f.SetCellValue(sheetName, cellName(6, row), float64(stat.MemVMS)/(1024.0*1024.0))
		f.SetCellValue(sheetName, cellName(7, row), stat.MemPercent)
		f.SetCellValue(sheetName, cellName(8, row), stat.Status)

		// Style cells
		f.SetCellStyle(sheetName, cellName(4, row), cellName(4, row), ids.percent)
		f.SetCellStyle(sheetName, cellName(5, row), cellName(6, row), ids.number)
		f.SetCellStyle(sheetName, cellName(7, row), cellName(7, row), ids.percent)

		if isAlt {
			f.SetCellStyle(sheetName, cellName(2, row), cellName(3, row), ids.altRow)
			f.SetCellStyle(sheetName, cellName(8, row), cellName(8, row), ids.altRow)
		}
	}

	// Summary stats at bottom
	lastDataRow := len(stats) + 2
	summRow := lastDataRow + 2

	// Average row
	f.SetCellValue(sheetName, cellName(1, summRow), "Average:")
	f.SetCellFormula(sheetName, cellName(4, summRow), fmt.Sprintf("=AVERAGE(D3:D%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(5, summRow), fmt.Sprintf("=AVERAGE(E3:E%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(6, summRow), fmt.Sprintf("=AVERAGE(F3:F%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(7, summRow), fmt.Sprintf("=AVERAGE(G3:G%d)", lastDataRow))

	// Max row
	f.SetCellValue(sheetName, cellName(1, summRow+1), "Max:")
	f.SetCellFormula(sheetName, cellName(4, summRow+1), fmt.Sprintf("=MAX(D3:D%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(5, summRow+1), fmt.Sprintf("=MAX(E3:E%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(6, summRow+1), fmt.Sprintf("=MAX(F3:F%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(7, summRow+1), fmt.Sprintf("=MAX(G3:G%d)", lastDataRow))

	// Min row
	f.SetCellValue(sheetName, cellName(1, summRow+2), "Min:")
	f.SetCellFormula(sheetName, cellName(4, summRow+2), fmt.Sprintf("=MIN(D3:D%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(5, summRow+2), fmt.Sprintf("=MIN(E3:E%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(6, summRow+2), fmt.Sprintf("=MIN(F3:F%d)", lastDataRow))
	f.SetCellFormula(sheetName, cellName(7, summRow+2), fmt.Sprintf("=MIN(G3:G%d)", lastDataRow))

	// Column widths
	colWidths := map[string]float64{
		"A": 22, "B": 9, "C": 20, "D": 10, "E": 16, "F": 16, "G": 11, "H": 12,
	}
	for col, w := range colWidths {
		f.SetColWidth(sheetName, col, col, w)
	}

	// Freeze top rows
	f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      2,
		TopLeftCell: "A3",
		ActivePane:  "bottomLeft",
	})

	return nil
}

// --- helpers ---

type summary struct {
	pid    int32
	avgCPU float64
	maxCPU float64
	avgRSS uint64
	maxRSS uint64
	avgMem float64
}

func computeSummary(stats []model.ProcessStat) summary {
	if len(stats) == 0 {
		return summary{}
	}
	s := summary{pid: stats[0].PID}
	for _, st := range stats {
		s.avgCPU += st.CPUPercent
		s.avgMem += float64(st.MemPercent)
		s.avgRSS += st.MemRSS
		if st.CPUPercent > s.maxCPU {
			s.maxCPU = st.CPUPercent
		}
		if st.MemRSS > s.maxRSS {
			s.maxRSS = st.MemRSS
		}
	}
	n := float64(len(stats))
	s.avgCPU /= n
	s.avgMem /= n
	s.avgRSS = uint64(float64(s.avgRSS) / n)
	return s
}

func groupByProcess(stats []model.ProcessStat) map[string][]model.ProcessStat {
	m := make(map[string][]model.ProcessStat)
	for _, s := range stats {
		key := fmt.Sprintf("%s_%d", s.Name, s.PID)
		m[key] = append(m[key], s)
	}
	return m
}

func sanitizeSheetName(name string) string {
	// Excel sheet names: max 31 chars, no special chars
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", "?", "_", "*", "_",
		"[", "_", "]", "_", ":", "_",
	)
	s := replacer.Replace(name)
	if len(s) > 28 {
		s = s[:28]
	}
	return s
}

func cellName(col, row int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)
	return c
}

func strPtr(s string) *string {
	return &s
}

package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// Column describes one column of a spreadsheet export.
type Column struct {
	Header string
	// CSVHeader replaces Header in the CSV output; empty means Header.
	CSVHeader string
	// Width is the XLSX column width; zero keeps the default.
	Width float64
	// Money formats the column as a value in reais: two decimals in the CSV,
	// number format 4 (#,##0.00) in the XLSX.
	Money bool
	// Center centers the XLSX data cells.
	Center bool
}

// Sheet is one XLSX worksheet: its name, the name of the table over its
// cells and its rows. Each cell is a string, an int or a float64, one per
// column.
type Sheet struct {
	Name  string
	Table string
	Rows  [][]any
}

// Spreadsheet is a CSV or XLSX export. WriteCSV writes every sheet under a
// single header; WriteXLSX writes one worksheet per sheet that has rows.
type Spreadsheet struct {
	Columns []Column
	// StatusCol is the index of the column holding the situação, used to
	// color the XLSX rows; -1 means none.
	StatusCol int
	Sheets    []Sheet
}

// validate rejects a row whose cell count does not match the columns.
func (s Spreadsheet) validate() error {
	for _, sheet := range s.Sheets {
		for i, row := range sheet.Rows {
			if len(row) != len(s.Columns) {
				return fmt.Errorf("sheet %q row %d has %d cells, want %d", sheet.Name, i, len(row), len(s.Columns))
			}
		}
	}
	return nil
}

// WriteCSV writes the rows of every sheet, in order, under one header line.
func WriteCSV(s Spreadsheet, outPath string) (err error) {
	if err := s.validate(); err != nil {
		return err
	}

	file, err := os.Create(outPath) // #nosec G304 -- destination is explicitly selected by the local user.
	if err != nil {
		return fmt.Errorf("failed to create csv file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close csv file: %w", cerr)
		}
	}()

	writer := csv.NewWriter(file)

	header := make([]string, len(s.Columns))
	for i, col := range s.Columns {
		header[i] = col.CSVHeader
		if header[i] == "" {
			header[i] = col.Header
		}
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, sheet := range s.Sheets {
		for _, row := range sheet.Rows {
			record := make([]string, len(row))
			for i, cell := range row {
				record[i] = csvCell(cell, s.Columns[i].Money)
			}
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("failed to write csv row: %w", err)
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to write csv file: %w", err)
	}
	return nil
}

// csvCell formats one cell for the CSV: money with two decimals, other
// numbers in their shortest form.
func csvCell(cell any, money bool) string {
	switch v := cell.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		if money {
			return strconv.FormatFloat(v, 'f', 2, 64)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

// WriteXLSX writes one worksheet per sheet with rows, each holding a styled
// table. When every sheet is empty the workbook has a single empty
// "Documentos" worksheet.
func WriteXLSX(s Spreadsheet, outPath string) error {
	if err := s.validate(); err != nil {
		return err
	}

	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	styles, err := newXLSXStyles(f)
	if err != nil {
		return err
	}

	// The new workbook starts with one worksheet: the first sheet with rows
	// takes it over, the others are added after it.
	defaultSheet := f.GetSheetName(0)
	wrote := false
	for _, sheet := range s.Sheets {
		if len(sheet.Rows) == 0 {
			continue
		}
		if !wrote {
			err = f.SetSheetName(defaultSheet, sheet.Name)
		} else {
			_, err = f.NewSheet(sheet.Name)
		}
		if err != nil {
			return fmt.Errorf("failed to create sheet %s: %w", sheet.Name, err)
		}
		if err := writeXLSXSheet(f, s, sheet, styles); err != nil {
			return err
		}
		wrote = true
	}
	if !wrote {
		if err := f.SetSheetName(defaultSheet, "Documentos"); err != nil {
			return fmt.Errorf("failed to create sheet Documentos: %w", err)
		}
	}

	if err := f.SaveAs(outPath); err != nil {
		return fmt.Errorf("failed to save excel file: %w", err)
	}
	return nil
}

type xlsxStyles struct {
	header, center, money int
	cancelled, faded      int
}

func newXLSXStyles(f *excelize.File) (xlsxStyles, error) {
	var st xlsxStyles
	var err error
	if st.money, err = f.NewStyle(&excelize.Style{NumFmt: 4}); err != nil {
		return st, fmt.Errorf("failed to create style: %w", err)
	}
	if st.header, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "top", WrapText: true},
	}); err != nil {
		return st, fmt.Errorf("failed to create style: %w", err)
	}
	if st.center, err = f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center"},
	}); err != nil {
		return st, fmt.Errorf("failed to create style: %w", err)
	}
	if st.cancelled, err = f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#C00000"},
	}); err != nil {
		return st, fmt.Errorf("failed to create cancelled conditional style: %w", err)
	}
	if st.faded, err = f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "#666666", Italic: true},
	}); err != nil {
		return st, fmt.Errorf("failed to create substituted conditional style: %w", err)
	}
	return st, nil
}

// writeXLSXSheet fills an existing worksheet with the header, the rows, the
// column styles and widths, the table and the situação coloring.
func writeXLSXSheet(f *excelize.File, s Spreadsheet, sheet Sheet, st xlsxStyles) error {
	name := sheet.Name
	lastRow := len(sheet.Rows) + 1

	for c, col := range s.Columns {
		cell, err := excelize.CoordinatesToCellName(c+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(name, cell, col.Header); err != nil {
			return fmt.Errorf("failed to write header on %s: %w", name, err)
		}
		if err := f.SetCellStyle(name, cell, cell, st.header); err != nil {
			return fmt.Errorf("failed to style header on %s: %w", name, err)
		}
	}

	for r, row := range sheet.Rows {
		for c, value := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(name, cell, value); err != nil {
				return fmt.Errorf("failed to write cell %s on %s: %w", cell, name, err)
			}
		}
	}

	for c, col := range s.Columns {
		letter, err := excelize.ColumnNumberToName(c + 1)
		if err != nil {
			return err
		}
		style := 0
		switch {
		case col.Money:
			style = st.money
		case col.Center:
			style = st.center
		}
		if style != 0 {
			if err := f.SetCellStyle(name, letter+"2", letter+strconv.Itoa(lastRow), style); err != nil {
				return fmt.Errorf("failed to style column %s on %s: %w", letter, name, err)
			}
		}
		if col.Width > 0 {
			if err := f.SetColWidth(name, letter, letter, col.Width); err != nil {
				return fmt.Errorf("failed to set width %s on %s: %w", letter, name, err)
			}
		}
	}

	lastCell, err := excelize.CoordinatesToCellName(len(s.Columns), lastRow)
	if err != nil {
		return err
	}
	if err := f.AddTable(name, &excelize.Table{
		Range:     "A1:" + lastCell,
		Name:      sheet.Table,
		StyleName: "TableStyleLight15",
	}); err != nil {
		return fmt.Errorf("failed to add table: %w", err)
	}

	if s.StatusCol < 0 {
		return nil
	}
	status, err := excelize.ColumnNumberToName(s.StatusCol + 1)
	if err != nil {
		return err
	}
	err = f.SetConditionalFormat(name, "A2:"+lastCell, []excelize.ConditionalFormatOptions{
		{
			Type:     "formula",
			Criteria: fmt.Sprintf(`=$%s2="cancelada"`, status),
			Format:   &st.cancelled,
		},
		{
			Type:     "formula",
			Criteria: fmt.Sprintf(`=OR($%[1]s2="substituida",$%[1]s2="denegada")`, status),
			Format:   &st.faded,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set conditional format on %s: %w", name, err)
	}
	return nil
}

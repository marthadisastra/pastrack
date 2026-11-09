package platform

import (
	"encoding/csv"
	"fmt"
	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
	"io"
	"os"
	"strings"
	"time"
)

// Convert a bounded SQL report without inventing values or interpreting sensor units.
func convertReport(source, destination, format string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if format == "xlsx" {
		book := excelize.NewFile()
		defer book.Close()
		book.SetSheetName("Sheet1", "Telemetry")
		index := 1
		for {
			record, e := reader.Read()
			if e == io.EOF {
				break
			}
			if e != nil {
				return e
			}
			values := make([]any, len(record))
			for i, v := range record {
				values[i] = v
			}
			if e = book.SetSheetRow("Telemetry", fmt.Sprintf("A%d", index), &values); e != nil {
				return e
			}
			index++
		}
		if err = book.SaveAs(destination); err != nil {
			return err
		}
		return os.Chmod(destination, 0600)
	}
	if format != "pdf" {
		return fmt.Errorf("unsupported report format")
	}
	headings, err := reader.Read()
	if err != nil {
		return err
	}
	if len(headings) == 0 {
		return fmt.Errorf("empty report schema")
	}
	font, err := assets.ReadFile("ui/fonts/inter-400.ttf")
	if err != nil {
		return err
	}
	pdf := fpdf.New("L", "mm", "A3", "")
	pdf.AddUTF8FontFromBytes("Inter", "", font)
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(false, 12)
	pdf.SetTitle("PasTrack Fleet Telemetry", false)
	pdf.AliasNbPages("")
	width := 396.0 / float64(len(headings))
	lineHeight := 3.2
	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Inter", "", 7)
		pdf.SetTextColor(115, 118, 134)
		pdf.CellFormat(396, 4, fmt.Sprintf("PasTrack · UTC timestamps · Halaman %d / {nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	page := func() {
		pdf.AddPage()
		pdf.SetFont("Inter", "", 16)
		pdf.SetTextColor(0, 74, 198)
		pdf.CellFormat(396, 9, "PasTrack — Fleet Telemetry Report", "", 1, "L", false, 0, "")
		pdf.SetFont("Inter", "", 8)
		pdf.SetTextColor(67, 70, 85)
		pdf.CellFormat(396, 6, "Generated "+time.Now().UTC().Format(time.RFC3339)+" · Missing values remain blank", "", 1, "L", false, 0, "")
		pdf.Ln(3)
		pdf.SetFont("Inter", "", 6)
		pdf.SetFillColor(37, 99, 235)
		pdf.SetTextColor(255, 255, 255)
		y := pdf.GetY()
		for i, h := range headings {
			pdf.SetXY(12+float64(i)*width, y)
			pdf.Rect(12+float64(i)*width, y, width, 14, "F")
			pdf.MultiCell(width, lineHeight, strings.ReplaceAll(h, "_", " "), "", "L", false)
		}
		pdf.SetXY(12, y+14)
		pdf.SetTextColor(25, 28, 30)
	}
	page()
	count := 0
	for {
		record, e := reader.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		count++
		if count > 5000 {
			return fmt.Errorf("PDF report exceeds 5000 rows")
		}
		height := lineHeight + 3
		for _, text := range record {
			h := float64(len(pdf.SplitText(text, width-3)))*lineHeight + 3
			if h > height {
				height = h
			}
		}
		if height > 230 {
			return fmt.Errorf("report cell too large for PDF page")
		}
		if pdf.GetY()+height > 282 {
			page()
		}
		y := pdf.GetY()
		for i, text := range record {
			x := 12 + float64(i)*width
			pdf.SetXY(x+1, y+1)
			pdf.SetDrawColor(224, 227, 229)
			pdf.Rect(x, y, width, height, "D")
			pdf.MultiCell(width-3, lineHeight, text, "", "L", false)
		}
		pdf.SetXY(12, y+height)
	}
	if err = pdf.OutputFileAndClose(destination); err != nil {
		return err
	}
	return os.Chmod(destination, 0600)
}

package platform

import (
	"github.com/xuri/excelize/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActualXLSXAndPDFExports(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.csv")
	if e := os.WriteFile(source, []byte("plate,fuel_liters,physical_ignition\nQA-ONLY,,NOT_INSTALLED\n"), 0600); e != nil {
		t.Fatal(e)
	}
	xlsx := filepath.Join(dir, "output.xlsx")
	if e := convertReport(source, xlsx, "xlsx"); e != nil {
		t.Fatal(e)
	}
	book, e := excelize.OpenFile(xlsx)
	if e != nil {
		t.Fatal(e)
	}
	defer book.Close()
	value, e := book.GetCellValue("Telemetry", "B2")
	if e != nil || value != "" {
		t.Fatal("missing fuel changed", value, e)
	}
	pdf := filepath.Join(dir, "output.pdf")
	if e = convertReport(source, pdf, "pdf"); e != nil {
		t.Fatal(e)
	}
	bytes, e := os.ReadFile(pdf)
	if e != nil || !strings.HasPrefix(string(bytes), "%PDF-") {
		t.Fatal("invalid PDF", e)
	}
}

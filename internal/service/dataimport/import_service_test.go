package dataimport

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"TagMatrix/internal/model"

	"github.com/xuri/excelize/v2"
)

// 创建一个临时的 SQLite DB 用于测试
func setupTestDB(t *testing.T, dbName string) string {
	dbPath := filepath.Join(os.TempDir(), dbName)
	_ = os.Remove(dbPath)

	err := model.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	return dbPath
}

// 清理临时文件
func teardownTestEnv(dbPath string, files ...string) {
	_ = os.Remove(dbPath)
	for _, f := range files {
		_ = os.Remove(f)
	}
}

// 创建测试用 CSV 文件
func createTestCSV(t *testing.T) string {
	filePath := filepath.Join(os.TempDir(), "test_import.csv")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	records := [][]string{
		{"id", "name", "age", "city"},
		{"1", "Alice", "25", "Beijing"},
		{"2", "Bob", "30", "Shanghai"},
		{"3", "Charlie", "35", "Guangzhou"},
	}

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			t.Fatalf("Failed to write to test CSV: %v", err)
		}
	}
	return filePath
}

// 创建测试用 Excel 文件
func createTestExcel(t *testing.T) string {
	filePath := filepath.Join(os.TempDir(), "test_import.xlsx")
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "id")
	f.SetCellValue(sheet, "B1", "name")
	f.SetCellValue(sheet, "C1", "age")
	f.SetCellValue(sheet, "D1", "city")

	f.SetCellValue(sheet, "A2", "1")
	f.SetCellValue(sheet, "B2", "Alice")
	f.SetCellValue(sheet, "C2", "25")
	f.SetCellValue(sheet, "D2", "Beijing")

	f.SetCellValue(sheet, "A3", "2")
	f.SetCellValue(sheet, "B3", "Bob")
	f.SetCellValue(sheet, "C3", "30")
	f.SetCellValue(sheet, "D3", "Shanghai")

	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("Failed to save test Excel: %v", err)
	}
	return filePath
}

func TestImportService_ImportCSV(t *testing.T) {
	dbPath := setupTestDB(t, "test_csv_import.db")
	csvPath := createTestCSV(t)
	defer teardownTestEnv(dbPath, csvPath)

	svc := &DataImportService{db: model.DB}

	count, err := svc.ImportData(csvPath)
	if err != nil {
		t.Fatalf("ImportData (CSV) failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 records imported, got %d", count)
	}

	// 验证数据库中是否有记录
	var records []model.RawDataRecord
	model.DB.Find(&records)

	if len(records) != 3 {
		t.Errorf("Expected 3 records in DB, got %d", len(records))
	}
}

func TestImportService_ExportCSV(t *testing.T) {
	dbPath := setupTestDB(t, "test_csv_export.db")
	csvPath := createTestCSV(t)
	exportPath := filepath.Join(os.TempDir(), "test_export.csv")
	defer teardownTestEnv(dbPath, csvPath, exportPath)

	svc := &DataImportService{db: model.DB}

	_, err := svc.ImportData(csvPath)
	if err != nil {
		t.Fatalf("ImportData failed: %v", err)
	}

	err = svc.ExportData(0, exportPath)
	if err != nil {
		t.Fatalf("ExportData failed: %v", err)
	}

	// 简单验证导出的文件是否存在
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Errorf("Exported CSV file does not exist")
	}
}

func TestImportService_ExportExcel(t *testing.T) {
	dbPath := setupTestDB(t, "test_excel_export.db")
	csvPath := createTestCSV(t)
	exportPath := filepath.Join(os.TempDir(), "test_export.xlsx")
	defer teardownTestEnv(dbPath, csvPath, exportPath)

	svc := &DataImportService{db: model.DB}

	_, err := svc.ImportData(csvPath)
	if err != nil {
		t.Fatalf("ImportData failed: %v", err)
	}

	err = svc.ExportData(0, exportPath)
	if err != nil {
		t.Fatalf("ExportData failed: %v", err)
	}

	// 简单验证导出的文件是否存在
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Errorf("Exported Excel file does not exist")
	}
}

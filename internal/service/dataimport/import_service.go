package dataimport

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"TagMatrix/internal/model"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// DataImportService 处理数据导入导出逻辑
type DataImportService struct {
	db *gorm.DB
}

// NewDataImportService 创建 DataImportService 实例
func NewDataImportService() *DataImportService {
	return &DataImportService{
		db: model.DB,
	}
}

// ExportData 根据指定条件导出数据到文件
func (s *DataImportService) ExportData(batchID uint64, exportPath string) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	ext := strings.ToLower(filepath.Ext(exportPath))
	if ext != ".csv" && ext != ".xlsx" {
		return fmt.Errorf("unsupported export format: %s", ext)
	}

	var records []model.RawDataRecord
	query := s.db.Model(&model.RawDataRecord{})
	if batchID > 0 {
		query = query.Where("batch_id = ?", batchID)
	}

	if err := query.Find(&records).Error; err != nil {
		return fmt.Errorf("failed to fetch records for export: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no data to export")
	}

	// 提取表头和数据行
	var headers []string
	var rows [][]string
	headerMap := make(map[string]bool)

	for _, record := range records {
		var dataMap map[string]interface{}
		if err := json.Unmarshal([]byte(record.Data), &dataMap); err != nil {
			continue // 忽略解析错误的数据
		}

		// 收集所有出现过的列作为表头
		for k := range dataMap {
			if !headerMap[k] {
				headerMap[k] = true
				headers = append(headers, k)
			}
		}
	}

	// 构建数据行
	for _, record := range records {
		var dataMap map[string]interface{}
		_ = json.Unmarshal([]byte(record.Data), &dataMap)

		row := make([]string, len(headers))
		for i, h := range headers {
			if val, ok := dataMap[h]; ok {
				row[i] = fmt.Sprintf("%v", val)
			} else {
				row[i] = ""
			}
		}
		rows = append(rows, row)
	}

	if ext == ".csv" {
		return s.exportToCSV(exportPath, headers, rows)
	}
	return s.exportToExcel(exportPath, headers, rows)
}

func (s *DataImportService) exportToCSV(filePath string, headers []string, rows [][]string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (s *DataImportService) exportToExcel(filePath string, headers []string, rows [][]string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"

	// 写入表头
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	// 写入数据
	for rIdx, row := range rows {
		for cIdx, val := range row {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rIdx+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	return f.SaveAs(filePath)
}

// AnalyzeFile 分析文件，返回基础信息和 Sheet 列表
func (s *DataImportService) AnalyzeFile(filePath string) (*model.FileAnalysisResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	fileName := filepath.Base(filePath)

	result := &model.FileAnalysisResult{
		FilePath: filePath,
		FileName: fileName,
	}

	if ext == ".csv" {
		result.FileType = "csv"
		return result, nil
	} else if ext == ".xlsx" || ext == ".xls" {
		result.FileType = "excel"
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open excel file: %w", err)
		}
		defer f.Close()

		result.SheetNames = f.GetSheetList()
		return result, nil
	}

	return nil, fmt.Errorf("unsupported file format: %s", ext)
}

// ImportData 导入 Excel 或 CSV 文件
// 返回导入的记录数和错误
func (s *DataImportService) ImportData(filePath string, selectedSheets []string) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var records []map[string]interface{}
	var err error

	switch ext {
	case ".csv":
		records, err = s.parseCSV(filePath)
	case ".xlsx", ".xls":
		records, err = s.parseExcel(filePath, selectedSheets)
	default:
		return 0, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to parse file: %w", err)
	}

	if len(records) == 0 {
		return 0, fmt.Errorf("no data found in file")
	}

	// 批量插入数据库
	return s.batchInsertRecords(records)
}

// parseCSV 解析 CSV 文件
func (s *DataImportService) parseCSV(filePath string) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// 读取表头
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv headers: %w", err)
	}

	var records []map[string]interface{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read csv row: %w", err)
		}

		record := make(map[string]interface{})
		for i, value := range row {
			if i < len(headers) {
				record[headers[i]] = value
			}
		}
		// 为 CSV 也附加来源表（文件名）
		record["来源表"] = filepath.Base(filePath)
		records = append(records, record)
	}

	return records, nil
}

// parseExcel 解析 Excel 文件
func (s *DataImportService) parseExcel(filePath string, selectedSheets []string) ([]map[string]interface{}, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheetsToProcess := selectedSheets
	// 如果前端没有传 sheet（比如直接导入模式或者默认情况），取第一个 sheet
	if len(sheetsToProcess) == 0 {
		sheetName := f.GetSheetName(0)
		if sheetName == "" {
			return nil, fmt.Errorf("no sheet found in excel file")
		}
		sheetsToProcess = append(sheetsToProcess, sheetName)
	}

	var allRecords []map[string]interface{}

	for _, sheetName := range sheetsToProcess {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			// 如果某个 sheet 读取失败，记录日志并继续，或者直接返回错误。这里选择返回错误以保证数据完整性
			return nil, fmt.Errorf("failed to read rows from sheet %s: %w", sheetName, err)
		}

		if len(rows) < 2 {
			continue // 跳过没有数据行的 sheet
		}

		headers := rows[0]

		for i := 1; i < len(rows); i++ {
			row := rows[i]
			record := make(map[string]interface{})
			for j, value := range row {
				if j < len(headers) {
					record[headers[j]] = value
				}
			}
			// 补齐空列
			for j := len(row); j < len(headers); j++ {
				record[headers[j]] = ""
			}

			// 可以在记录中附加来源 sheet 名，方便追溯
			record["来源表"] = sheetName

			allRecords = append(allRecords, record)
		}
	}

	return allRecords, nil
}

// batchInsertRecords 批量将数据序列化并入库
func (s *DataImportService) batchInsertRecords(records []map[string]interface{}) (int, error) {
	// 生成一个统一的批次 ID，这里简单使用时间戳，实际中可以使用 UUID
	batchID := uint64(time.Now().UnixNano())

	// 分批插入，避免内存或 SQL 语句过大
	batchSize := 1000
	insertedCount := 0

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var batchData []model.RawDataRecord

		for i, record := range records {
			jsonData, err := json.Marshal(record)
			if err != nil {
				return fmt.Errorf("failed to marshal record to JSON: %w", err)
			}

			batchData = append(batchData, model.RawDataRecord{
				BatchID: batchID,
				Data:    string(jsonData),
			})

			if len(batchData) >= batchSize || i == len(records)-1 {
				if err := tx.Create(&batchData).Error; err != nil {
					return fmt.Errorf("failed to batch insert records: %w", err)
				}
				insertedCount += len(batchData)
				batchData = batchData[:0] // 清空切片
			}
		}
		return nil
	})

	return insertedCount, err
}

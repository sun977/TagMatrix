package dataadmin

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type DataAdminService struct {
	db *gorm.DB
}

func NewDataAdminService(db *gorm.DB) *DataAdminService {
	return &DataAdminService{
		db: db,
	}
}

type RawSQLResult struct {
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	Affected int64                    `json:"affected"`
	Duration string                   `json:"duration"`
	IsSelect bool                     `json:"is_select"`
}

// ExecuteRawSQL 执行原始 SQL
func (s *DataAdminService) ExecuteRawSQL(query string) (*RawSQLResult, error) {
	start := time.Now()
	res := &RawSQLResult{
		IsSelect: strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") || strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "PRAGMA"),
	}

	if res.IsSelect {
		rows, err := s.db.Raw(query).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		res.Columns = cols

		for rows.Next() {
			// Scan row
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range cols {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, err
			}

			rowMap := make(map[string]interface{})
			for i, col := range cols {
				var v interface{}
				val := values[i]
				b, ok := val.([]byte)
				if ok {
					v = string(b)
				} else {
					v = val
				}
				rowMap[col] = v
			}
			res.Rows = append(res.Rows, rowMap)
		}
	} else {
		tx := s.db.Exec(query)
		if tx.Error != nil {
			return nil, tx.Error
		}
		res.Affected = tx.RowsAffected
	}

	res.Duration = time.Since(start).String()
	return res, nil
}

// GetSystemTables 获取物理表
func (s *DataAdminService) GetSystemTables() ([]string, error) {
	var tables []string
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	if err := s.db.Raw(query).Scan(&tables).Error; err != nil {
		return nil, err
	}
	return tables, nil
}

type PagedTableData struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Total   int64                    `json:"total"`
}

// GetTableData 获取物理表分页数据
func (s *DataAdminService) GetTableData(tableName string, offset, limit int) (*PagedTableData, error) {
	res := &PagedTableData{}

	// get total
	var total int64
	if err := s.db.Table(tableName).Count(&total).Error; err != nil {
		return nil, err
	}
	res.Total = total

	// get rows
	rows, err := s.db.Raw(fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", tableName), limit, offset).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	res.Columns = cols

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		rowMap := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		res.Rows = append(res.Rows, rowMap)
	}

	return res, nil
}

// GetVirtualDatasetData 获取虚拟业务数据集分页数据
func (s *DataAdminService) GetVirtualDatasetData(datasetId uint, offset, limit int) (*PagedTableData, error) {
	res := &PagedTableData{
		Columns: []string{"id"},
	}

	var total int64
	if err := s.db.Table("raw_data_records").Where("dataset_id = ?", datasetId).Count(&total).Error; err != nil {
		return nil, err
	}
	res.Total = total

	type RawData struct {
		ID   uint   `gorm:"column:id"`
		Data string `gorm:"column:data"` // json field
	}

	var records []RawData
	if err := s.db.Table("raw_data_records").Where("dataset_id = ?", datasetId).Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, err
	}

	keySet := make(map[string]bool)

	for _, rec := range records {
		var j map[string]interface{}
		if err := json.Unmarshal([]byte(rec.Data), &j); err != nil {
			continue
		}
		rowMap := make(map[string]interface{})
		rowMap["id"] = rec.ID
		for k, v := range j {
			if !keySet[k] {
				res.Columns = append(res.Columns, k)
				keySet[k] = true
			}
			rowMap[k] = v
		}
		res.Rows = append(res.Rows, rowMap)
	}

	return res, nil
}

// UpdateVirtualRecord 更新虚拟记录
func (s *DataAdminService) UpdateVirtualRecord(recordId uint, payload map[string]interface{}) error {
	type RawData struct {
		ID   uint   `gorm:"column:id"`
		Data string `gorm:"column:data"`
	}

	var rec RawData
	if err := s.db.Table("raw_data_records").Where("id = ?", recordId).First(&rec).Error; err != nil {
		return err
	}

	var j map[string]interface{}
	if err := json.Unmarshal([]byte(rec.Data), &j); err != nil {
		j = make(map[string]interface{})
	}

	for k, v := range payload {
		j[k] = v
	}

	newData, err := json.Marshal(j)
	if err != nil {
		return err
	}

	return s.db.Table("raw_data_records").Where("id = ?", recordId).Update("data", string(newData)).Error
}

// DeleteVirtualRecord 删除虚拟记录
func (s *DataAdminService) DeleteVirtualRecord(recordId uint) error {
	return s.db.Table("raw_data_records").Where("id = ?", recordId).Delete(nil).Error
}

type SysSqlTemplate struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Query string `json:"query"`
}

// GetSqlTemplates 获取所有保存的SQL模板
func (s *DataAdminService) GetSqlTemplates() ([]SysSqlTemplate, error) {
	var templates []SysSqlTemplate
	err := s.db.Table("sys_sql_templates").Select("id, name, query").Find(&templates).Error
	return templates, err
}

// SaveSqlTemplate 保存或更新SQL模板
func (s *DataAdminService) SaveSqlTemplate(id uint64, name, query string) error {
	if name == "" || query == "" {
		return fmt.Errorf("模板名称和查询语句不能为空")
	}

	if id > 0 {
		return s.db.Table("sys_sql_templates").Where("id = ?", id).Updates(map[string]interface{}{
			"name":  name,
			"query": query,
		}).Error
	}

	return s.db.Table("sys_sql_templates").Create(map[string]interface{}{
		"name":  name,
		"query": query,
	}).Error
}

// DeleteSqlTemplate 删除SQL模板
func (s *DataAdminService) DeleteSqlTemplate(id uint64) error {
	return s.db.Table("sys_sql_templates").Where("id = ?", id).Delete(nil).Error
}

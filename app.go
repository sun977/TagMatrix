package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/service/aiengine"
	"TagMatrix/internal/service/dataimport"
	"TagMatrix/internal/service/dataset"
	"TagMatrix/internal/service/taglogic"
	"TagMatrix/internal/service/taskengine"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	dataset    *dataset.DatasetService
	dataImport *dataimport.DataImportService
	tagLogic   *taglogic.TagLogicService
	taskEngine *taskengine.TaskEngineService
	aiEngine   *aiengine.AIEngineService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 确定数据存放目录
	var appDir string
	env := runtime.Environment(ctx)
	if env.BuildType == "dev" || env.BuildType == "debug" {
		// 开发模式：直接使用当前项目根目录
		appDir = "."
		fmt.Println("Running in Dev/Debug mode, using current directory for data and config.")
	} else {
		// 生产打包模式：使用系统标准的 AppData 目录
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = "."
		}
		appDir = filepath.Join(configDir, "TagMatrix")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			fmt.Printf("Failed to create app directory: %v\n", err)
			appDir = "."
		}
		fmt.Printf("Running in Production mode, using directory: %s\n", appDir)
	}

	dbPath := filepath.Join(appDir, "data.db")

	// 1. 初始化数据库
	err := model.InitDB(dbPath)
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
	}

	// 2. 初始化配置文件
	err = config.InitConfig(appDir)
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
	}

	// 3. 初始化所有的 Service
	a.dataset = dataset.NewDatasetService()
	a.dataImport = dataimport.NewDataImportService()
	a.tagLogic = taglogic.NewTagLogicService()
	a.taskEngine = taskengine.NewTaskEngineService(ctx)
	a.aiEngine = aiengine.NewAIEngineService()
}

// ----------------- Config API -----------------

// GetAppConfig 获取当前应用的配置
func (a *App) GetAppConfig() config.AppConfig {
	return config.GetConfig()
}

// SaveAppConfig 保存应用的配置
func (a *App) SaveAppConfig(newConfig config.AppConfig) error {
	return config.SaveConfig(newConfig)
}

// ----------------- Dashboard & Stats API -----------------

func (a *App) GetDashboardStats() (*model.DashboardStats, error) {
	var stats model.DashboardStats

	// 总数据量
	model.DB.Model(&model.RawDataRecord{}).Count(&stats.TotalRecords)

	// 标签总数
	model.DB.Model(&model.SysTag{}).Count(&stats.TotalTags)

	// 规则总数
	model.DB.Model(&model.SysMatchRule{}).Count(&stats.TotalRules)

	// 已打标数据量 (去重统计有多少 distinct record_id 在 sys_entity_tags 表中，且关联的记录未被软删除)
	model.DB.Model(&model.SysEntityTag{}).
		Joins("JOIN raw_data_records ON raw_data_records.id = sys_entity_tags.record_id").
		Where("raw_data_records.deleted_at IS NULL").
		Distinct("sys_entity_tags.record_id").
		Count(&stats.TaggedRecords)

	// 按数据集分组统计
	var datasets []model.SysDataset
	model.DB.Find(&datasets)

	for _, ds := range datasets {
		var dsStat model.DatasetStat
		dsStat.DatasetID = ds.ID
		dsStat.DatasetName = ds.Name

		model.DB.Model(&model.RawDataRecord{}).Where("dataset_id = ?", ds.ID).Count(&dsStat.TotalRecords)

		model.DB.Model(&model.SysEntityTag{}).
			Joins("JOIN raw_data_records ON raw_data_records.id = sys_entity_tags.record_id").
			Where("raw_data_records.deleted_at IS NULL AND raw_data_records.dataset_id = ?", ds.ID).
			Distinct("sys_entity_tags.record_id").
			Count(&dsStat.TaggedRecords)

		stats.DatasetStats = append(stats.DatasetStats, dsStat)
	}

	return &stats, nil
}

// ----------------- Dataset API -----------------

// ListDatasets 获取所有数据集
func (a *App) ListDatasets() ([]model.SysDataset, error) {
	return a.dataset.ListDatasets()
}

// CreateDataset 创建数据集
func (a *App) CreateDataset(name, description string) (*model.SysDataset, error) {
	return a.dataset.CreateDataset(name, description)
}

// UpdateDataset 更新数据集
func (a *App) UpdateDataset(id uint64, name, description string) error {
	return a.dataset.UpdateDataset(id, name, description)
}

// DeleteDataset 删除数据集
func (a *App) DeleteDataset(id uint64) error {
	return a.dataset.DeleteDataset(id)
}

// ----------------- Data Import/Export API -----------------

// AnalyzeDataFile 分析文件并返回表信息 (前端用来做多 Sheet 选择)
func (a *App) AnalyzeDataFile() (*model.FileAnalysisResult, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要导入的数据文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "表格文件", Pattern: "*.csv;*.xlsx"},
		},
	})
	if err != nil {
		return nil, err
	}
	if file == "" {
		return nil, fmt.Errorf("cancelled")
	}

	return a.dataImport.AnalyzeFile(file)
}

func (a *App) ImportData(filePath string, selectedSheets []string, datasetID uint64, newDatasetName string) (int, error) {
	return a.dataImport.ImportData(filePath, selectedSheets, datasetID, newDatasetName)
}

func (a *App) ExportData(datasetID uint64, exportPath string) error {
	if exportPath == "" {
		file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "选择导出路径",
			DefaultFilename: "export_data.csv",
			Filters: []runtime.FileFilter{
				{DisplayName: "CSV 文件", Pattern: "*.csv"},
				{DisplayName: "Excel 文件", Pattern: "*.xlsx"},
			},
		})
		if err != nil {
			return err
		}
		if file == "" {
			return fmt.Errorf("cancelled")
		}
		exportPath = file
	}
	return a.dataImport.ExportData(datasetID, exportPath)
}

type PagedData struct {
	Total   int64
	Records []model.RawDataRecord
}

func (a *App) GetRawDataList(datasetID uint64, page, pageSize int, searchCol, keyword string) (*PagedData, error) {
	var records []model.RawDataRecord
	var total int64

	db := model.DB.Model(&model.RawDataRecord{}).Where("dataset_id = ?", datasetID)

	if keyword != "" {
		if searchCol != "" {
			// 如果指定了具体列，使用 JSON_EXTRACT 进行模糊匹配
			// SQLite 语法: json_extract(data, '$.colName') LIKE '%keyword%'
			db = db.Where(fmt.Sprintf("json_extract(data, '$.%s') LIKE ?", searchCol), "%"+keyword+"%")
		} else {
			// 如果未指定列，全局搜索 JSON 字符串
			db = db.Where("data LIKE ?", "%"+keyword+"%")
		}
	}

	db.Count(&total)

	offset := (page - 1) * pageSize
	err := db.Order("id desc").Offset(offset).Limit(pageSize).Find(&records).Error
	if err != nil {
		return nil, err
	}

	return &PagedData{
		Total:   total,
		Records: records,
	}, nil
}

func (a *App) DeleteRawData(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	// GORM 软删除或硬删除，取决于 RawDataRecord 是否包含 gorm.DeletedAt
	return model.DB.Delete(&model.RawDataRecord{}, ids).Error
}

func (a *App) GetTaggedDataList(keyword, tag, batch, searchCol, dataSource, tagMode, status, startDate, endDate string, page, pageSize int) (*model.PagedTaggedData, error) {
	var total int64
	var dtos []model.TaggedRecordDto

	// 1. 构建查询构造器
	db := model.DB.Model(&model.RawDataRecord{})

	if keyword != "" {
		if searchCol != "" {
			// 根据指定列搜索
			db = db.Where("json_extract(raw_data_records.data, '$."+searchCol+"') LIKE ?", "%"+keyword+"%")
		} else {
			// 全局搜索
			db = db.Where("raw_data_records.data LIKE ?", "%"+keyword+"%")
		}
	}

	if dataSource != "" {
		db = db.Where("json_extract(raw_data_records.data, '$.\"数据来源\"') = ?", dataSource)
	}

	if startDate != "" {
		db = db.Where("raw_data_records.updated_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		db = db.Where("raw_data_records.updated_at <= ?", endDate+" 23:59:59")
	}

	// 其他关联过滤条件需要在连接表上操作，因为我们需要分页
	// 由于 Tag, Batch, TagMode, Status 是多对多或根据计算得出的，我们最好使用 Join 或者子查询

	if tag != "" || batch != "" || tagMode != "" || status != "" {
		subQuery := model.DB.Table("sys_entity_tags").Select("record_id")

		if tag != "" {
			subQuery = subQuery.Where("tag_id = ?", tag)
		}

		if batch != "" {
			subQuery = subQuery.Where("batch_id = ?", batch)
		}

		if tagMode != "" {
			// batch 表关联
			subQuery = subQuery.Joins("JOIN tag_task_batches ON tag_task_batches.id = sys_entity_tags.batch_id").
				Where("tag_task_batches.tag_mode = ?", tagMode)
		}

		switch status {
		case "success":
			db = db.Where("raw_data_records.id IN (?)", subQuery)
		case "unmatched":
			db = db.Where("raw_data_records.id NOT IN (?)", subQuery)
		default:
			db = db.Where("raw_data_records.id IN (?)", subQuery)
		}
	}

	// 2. 统计总数
	db.Count(&total)

	// 3. 分页查询原始记录
	var records []model.RawDataRecord
	offset := (page - 1) * pageSize
	err := db.Select("raw_data_records.*").Order("raw_data_records.id desc").Offset(offset).Limit(pageSize).Find(&records).Error
	if err != nil {
		return nil, err
	}

	// 4. 组装 DTO (查询相关的 Tags 和 Batch)
	for _, r := range records {
		dto := model.TaggedRecordDto{
			ID:         r.ID,
			Content:    r.Data, // 将原始数据内容传递给前端，前端可做解析
			UpdateTime: r.UpdatedAt.Format("2006-01-02 15:04:05"),
			Tags:       []model.TagDto{},
		}

		// 解析数据来源
		var dataMap map[string]any
		if err := json.Unmarshal([]byte(r.Data), &dataMap); err == nil {
			if src, ok := dataMap["数据来源"].(string); ok {
				dto.DataSource = src
			}
		}

		// 查询这条记录的所有标签
		var entityTags []model.SysEntityTag
		model.DB.Where("record_id = ?", r.ID).Find(&entityTags)

		if len(entityTags) > 0 {
			dto.Status = "success"
			// 查出最后一个 BatchID
			lastBatchID := entityTags[len(entityTags)-1].BatchID
			if lastBatchID > 0 {
				var b model.TagTaskBatch
				if err := model.DB.First(&b, lastBatchID).Error; err == nil {
					dto.BatchName = b.Name
					dto.TagMode = b.TagMode
				}
			}

			// 构建 TagID 到 IsPrimary 的映射
			primaryTagMap := make(map[uint64]bool)
			var tagIDs []uint64
			for _, et := range entityTags {
				tagIDs = append(tagIDs, et.TagID)
				if et.IsPrimary {
					primaryTagMap[et.TagID] = true
				}
			}

			if len(tagIDs) > 0 {
				var tags []model.SysTag
				model.DB.Where("id IN ?", tagIDs).Find(&tags)
				for _, t := range tags {
					tagDto := model.TagDto{
						Name:  t.Name,
						Color: t.Color,
					}
					dto.Tags = append(dto.Tags, tagDto)

					// 如果是主标签
					if primaryTagMap[t.ID] {
						dto.PrimaryTag = &tagDto
					}
				}
			}
		} else {
			dto.Status = "unmatched"
			dto.BatchName = "-"
		}

		dtos = append(dtos, dto)
	}

	return &model.PagedTaggedData{
		Total:   total,
		Records: dtos,
	}, nil
}

// ExportTaggedDataList 按筛选条件导出打标数据，包含动态字段和系统处理字段，不包含 ID 和打标时间
func (a *App) ExportTaggedDataList(keyword, tag, batch, searchCol, dataSource, tagMode, status, startDate, endDate string) error {
	// 构建查询条件
	db := model.DB.Model(&model.RawDataRecord{})

	if keyword != "" {
		if searchCol != "" {
			db = db.Where("json_extract(raw_data_records.data, '$."+searchCol+"') LIKE ?", "%"+keyword+"%")
		} else {
			db = db.Where("raw_data_records.data LIKE ?", "%"+keyword+"%")
		}
	}

	if dataSource != "" {
		db = db.Where("json_extract(raw_data_records.data, '$.\"数据来源\"') = ?", dataSource)
	}

	if startDate != "" {
		db = db.Where("raw_data_records.updated_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		db = db.Where("raw_data_records.updated_at <= ?", endDate+" 23:59:59")
	}

	if tag != "" || batch != "" || tagMode != "" || status != "" {
		subQuery := model.DB.Table("sys_entity_tags").Select("record_id")

		if tag != "" {
			subQuery = subQuery.Where("tag_id = ?", tag)
		}

		if batch != "" {
			subQuery = subQuery.Where("batch_id = ?", batch)
		}

		if tagMode != "" {
			subQuery = subQuery.Joins("JOIN tag_task_batches ON tag_task_batches.id = sys_entity_tags.batch_id").
				Where("tag_task_batches.tag_mode = ?", tagMode)
		}

		switch status {
		case "success":
			db = db.Where("raw_data_records.id IN (?)", subQuery)
		case "unmatched":
			db = db.Where("raw_data_records.id NOT IN (?)", subQuery)
		default:
			db = db.Where("raw_data_records.id IN (?)", subQuery)
		}
	}

	// 查出所有符合条件的原始记录
	var records []model.RawDataRecord
	err := db.Select("raw_data_records.*").Order("raw_data_records.id desc").Find(&records).Error
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return fmt.Errorf("没有找到符合条件的数据可供导出")
	}

	// 弹窗让用户选择保存位置
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "选择导出路径",
		DefaultFilename: "tagged_data_export.csv",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV 文件", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return err
	}
	if filePath == "" {
		return fmt.Errorf("cancelled")
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入 UTF-8 BOM，防止 Excel 乱码
	file.Write([]byte("\xEF\xBB\xBF"))
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 准备提取所有的动态列
	var dynamicCols []string
	colSet := make(map[string]bool)

	// 先遍历一次所有数据，收集所有的键
	parsedDataMap := make([]map[string]any, len(records))
	for i, r := range records {
		var dMap map[string]any
		json.Unmarshal([]byte(r.Data), &dMap)
		parsedDataMap[i] = dMap
		for k := range dMap {
			// 跳过 id 和 数据来源（数据来源放在最后固定位置）
			if !colSet[k] && k != "id" && k != "数据来源" {
				colSet[k] = true
				dynamicCols = append(dynamicCols, k)
			}
		}
	}

	// 构建表头 (不要系统 ID 和打标时间)
	headers := append([]string{}, dynamicCols...)
	headers = append(headers, "打标模式", "命中标签", "命中主标签", "任务批次", "数据来源", "状态")

	if err := writer.Write(headers); err != nil {
		return err
	}

	// 处理并写入每行数据
	for i, r := range records {
		dMap := parsedDataMap[i]

		// 查询关联信息
		var entityTags []model.SysEntityTag
		model.DB.Where("record_id = ?", r.ID).Find(&entityTags)

		statusVal := "未命中"
		batchName := "-"
		tagModeVal := "-"
		tagsStr := "-"
		primaryTagStr := "-"

		if len(entityTags) > 0 {
			statusVal = "已打标"
			lastBatchID := entityTags[len(entityTags)-1].BatchID
			if lastBatchID > 0 {
				var b model.TagTaskBatch
				if err := model.DB.First(&b, lastBatchID).Error; err == nil {
					batchName = b.Name
					tagModeVal = b.TagMode
				}
			}

			var tagIDs []uint64
			primaryMap := make(map[uint64]bool)
			for _, et := range entityTags {
				tagIDs = append(tagIDs, et.TagID)
				if et.IsPrimary {
					primaryMap[et.TagID] = true
				}
			}

			if len(tagIDs) > 0 {
				var tags []model.SysTag
				model.DB.Where("id IN ?", tagIDs).Find(&tags)

				var tNames []string
				for _, t := range tags {
					tNames = append(tNames, t.Name)
					if primaryMap[t.ID] {
						primaryTagStr = t.Name
					}
				}
				if len(tNames) > 0 {
					tagsStr = strings.Join(tNames, ", ")
				}
			}
		}

		// 格式化打标模式
		switch tagModeVal {
		case "single":
			tagModeVal = "单标签"
		case "multiple":
			tagModeVal = "多标签"
		case "mixed":
			tagModeVal = "混合模式"
		case "":
			tagModeVal = "-"
		}

		// 数据来源
		dataSourceStr := "-"
		if ds, ok := dMap["数据来源"].(string); ok {
			dataSourceStr = ds
		}

		// 构建单行数组
		row := make([]string, 0, len(headers))
		// 动态列值
		for _, k := range dynamicCols {
			if val, ok := dMap[k]; ok {
				row = append(row, fmt.Sprintf("%v", val))
			} else {
				row = append(row, "")
			}
		}
		// 追加系统处理字段
		row = append(row, tagModeVal, tagsStr, primaryTagStr, batchName, dataSourceStr, statusVal)

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// ----------------- Tag & Rule Logic API -----------------

func (a *App) CreateTag(tag model.SysTag) error {
	return a.tagLogic.CreateTag(&tag)
}

func (a *App) UpdateTag(tag model.SysTag) error {
	return a.tagLogic.UpdateTag(&tag)
}

func (a *App) DeleteTag(id uint64) error {
	return a.tagLogic.DeleteTag(id)
}

func (a *App) CheckTagHasRules(id uint64) (bool, error) {
	return a.tagLogic.CheckTagHasRules(id)
}

func (a *App) GetAllTags() ([]model.SysTag, error) {
	return a.tagLogic.GetAllTags()
}

func (a *App) GetTagTree() ([]model.TagTreeNode, error) {
	return a.tagLogic.GetTagTree()
}

func (a *App) ExportTags(exportPath string) error {
	if exportPath == "" {
		file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "导出标签结构",
			DefaultFilename: "tags_export.json",
			Filters: []runtime.FileFilter{
				{DisplayName: "JSON 文件", Pattern: "*.json"},
			},
		})
		if err != nil {
			return err
		}
		if file == "" {
			return fmt.Errorf("cancelled")
		}
		exportPath = file
	}
	return a.tagLogic.ExportTags(exportPath)
}

func (a *App) ImportTags(filePath string) error {
	if filePath == "" {
		file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
			Title: "导入标签结构",
			Filters: []runtime.FileFilter{
				{DisplayName: "JSON 文件", Pattern: "*.json"},
			},
		})
		if err != nil {
			return err
		}
		if file == "" {
			return fmt.Errorf("cancelled")
		}
		filePath = file
	}
	return a.tagLogic.ImportTags(filePath)
}

func (a *App) SaveRule(rule model.SysMatchRule) error {
	return a.tagLogic.SaveRule(&rule)
}

func (a *App) GetRulesByTag(tagID uint64) ([]model.SysMatchRule, error) {
	return a.tagLogic.GetRulesByTagID(tagID)
}

func (a *App) GetRulesByDataset(datasetID uint64) ([]model.SysMatchRule, error) {
	return a.tagLogic.GetRulesByDataset(datasetID)
}

func (a *App) DeleteRule(id uint64) error {
	return a.tagLogic.DeleteRule(id)
}

func (a *App) GetAllRules() ([]model.SysMatchRule, error) {
	var rules []model.SysMatchRule
	err := model.DB.Find(&rules).Error
	return rules, err
}

func (a *App) DryRunRule(ruleJSON string, limit int) ([]taglogic.DryRunResult, error) {
	return a.tagLogic.DryRunRule(ruleJSON, limit)
}

// ----------------- Task Engine API -----------------
func (a *App) RunTaggingTask(datasetID uint64, ruleIDs []uint64, batchName string, isOverwrite bool, tagMode string, dataSource string) (uint64, error) {
	return a.taskEngine.RunTaggingTask(datasetID, ruleIDs, batchName, isOverwrite, tagMode, dataSource)
}

// GetAvailableDataSources 获取所有可用的数据来源选项
func (a *App) GetAvailableDataSources(datasetID uint64) ([]model.DataSourceOption, error) {
	return a.taskEngine.GetAvailableDataSources(a.ctx, datasetID)
}

func (a *App) RollbackTask(batchID uint64) error {
	return a.taskEngine.RollbackTask(a.ctx, batchID)
}

func (a *App) DeleteTaskBatches(batchIDs []uint64) error {
	return a.taskEngine.DeleteTaskBatches(a.ctx, batchIDs)
}

func (a *App) GetTaskBatches() ([]model.TagTaskBatch, error) {
	var batches []model.TagTaskBatch
	err := model.DB.Order("id desc").Find(&batches).Error
	return batches, err
}

func (a *App) GetTaskLogs(batchID uint64) ([]model.TagTaskLogDto, error) {
	return a.taskEngine.GetTaskLogs(batchID)
}

// ExportTaskLogsCSV exports task logs to a CSV file selected by the user
func (a *App) ExportTaskLogsCSV(batchID uint64) (string, error) {
	// 1. Fetch logs
	logs, err := a.taskEngine.GetTaskLogs(batchID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch task logs: %v", err)
	}

	// 2. Ask user for file path
	defaultFilename := fmt.Sprintf("task_logs_%d.csv", batchID)
	filepath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Title:           "导出打标日志",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "CSV Files (*.csv)",
				Pattern:     "*.csv",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to open save dialog: %v", err)
	}
	if filepath == "" {
		// User cancelled
		return "", nil
	}

	// 3. Create file
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// 4. Write CSV (Adding BOM for Excel compatibility)
	file.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"数据ID", "标签名称", "命中规则", "操作", "匹配原因", "时间"}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write csv header: %v", err)
	}

	// Write rows
	for _, log := range logs {
		action := "移除"
		if log.Action == "add" {
			action = "添加"
		}

		row := []string{
			strconv.FormatUint(log.RecordID, 10),
			log.TagName,
			log.RuleName,
			action,
			log.Reason,
			log.CreatedAt,
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write csv row: %v", err)
		}
	}

	return filepath, nil
}

// ----------------- AI Engine API -----------------

func (a *App) ChatWithAI(message string) (string, error) {
	return a.aiEngine.ChatWithAI(a.ctx, message)
}

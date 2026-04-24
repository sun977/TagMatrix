package main

import (
	"context"
	"fmt"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/service/aiengine"
	"TagMatrix/internal/service/dataimport"
	"TagMatrix/internal/service/taglogic"
	"TagMatrix/internal/service/taskengine"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
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

	// 1. 初始化数据库
	// 实际环境中应将数据库保存在用户的 AppData 目录下，这里简单使用当前目录的 data.db
	err := model.InitDB("data.db")
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
	}

	// 2. 初始化配置文件 (保存到当前目录)
	err = config.InitConfig(".")
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
	}

	// 3. 初始化所有的 Service
	a.dataImport = dataimport.NewDataImportService()
	a.tagLogic = taglogic.NewTagLogicService()
	a.taskEngine = taskengine.NewTaskEngineService()
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

	// 已打标数据量 (去重统计有多少 record_id 在 entity_tag 表中)
	model.DB.Model(&model.SysEntityTag{}).Select("count(distinct(record_id))").Count(&stats.TaggedRecords)

	return &stats, nil
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

func (a *App) ImportData(filePath string, selectedSheets []string) (int, error) {
	return a.dataImport.ImportData(filePath, selectedSheets)
}

func (a *App) ExportData(batchID uint64, exportPath string) error {
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
	return a.dataImport.ExportData(batchID, exportPath)
}

type PagedData struct {
	Total   int64
	Records []model.RawDataRecord
}

func (a *App) GetRawDataList(page, pageSize int, searchCol, keyword string) (*PagedData, error) {
	var records []model.RawDataRecord
	var total int64

	db := model.DB.Model(&model.RawDataRecord{})

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

func (a *App) GetTaggedDataList(keyword, tag, batch string, page, pageSize int) (*model.PagedTaggedData, error) {
	var total int64
	var dtos []model.TaggedRecordDto

	// 1. 构建查询构造器
	db := model.DB.Model(&model.RawDataRecord{})

	// 如果有 tag 或者 batch 过滤，我们需要 JOIN sys_entity_tag 表
	if tag != "" || batch != "" {
		db = db.Joins("JOIN sys_entity_tags ON sys_entity_tags.record_id = raw_data_records.id")
		if tag != "" {
			// 根据 tag_id 过滤
			db = db.Where("sys_entity_tags.tag_id = ?", tag)
		}
		if batch != "" {
			// 根据 batch_id 过滤
			db = db.Where("sys_entity_tags.batch_id = ?", batch)
		}
		// 因为 join 可能会产生重复记录，我们需要 group by 原始记录 ID
		db = db.Group("raw_data_records.id")
	}

	if keyword != "" {
		db = db.Where("raw_data_records.data LIKE ?", "%"+keyword+"%")
	}

	// 2. 统计总数 (需要考虑 GROUP BY 的情况)
	if tag != "" || batch != "" {
		// 使用子查询统计总数
		model.DB.Table("(?) as u", db.Select("raw_data_records.id")).Count(&total)
	} else {
		db.Count(&total)
	}

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
				}
			}

			// 查询具体的 Tag 详情
			var tagIDs []uint64
			for _, et := range entityTags {
				tagIDs = append(tagIDs, et.TagID)
			}

			if len(tagIDs) > 0 {
				var tags []model.SysTag
				model.DB.Where("id IN ?", tagIDs).Find(&tags)
				for _, t := range tags {
					dto.Tags = append(dto.Tags, model.TagDto{
						Name:  t.Name,
						Color: t.Color,
					})
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

// ----------------- Tag & Rule Logic API -----------------

func (a *App) CreateTag(tag model.SysTag) error {
	return a.tagLogic.CreateTag(&tag)
}

func (a *App) GetAllTags() ([]model.SysTag, error) {
	return a.tagLogic.GetAllTags()
}

func (a *App) SaveRule(rule model.SysMatchRule) error {
	return a.tagLogic.SaveRule(&rule)
}

func (a *App) GetRuleByTag(tagID uint64) (*model.SysMatchRule, error) {
	var rule model.SysMatchRule
	err := model.DB.Where("tag_id = ?", tagID).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
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
func (a *App) RunTaggingTask(ruleIDs []uint64, batchName string, isPrimary bool) (uint64, error) {
	return a.taskEngine.RunTaggingTask(ruleIDs, batchName, isPrimary)
}

func (a *App) RollbackTask(batchID uint64) error {
	return a.taskEngine.RollbackTask(a.ctx, batchID)
}

func (a *App) GetTaskBatches() ([]model.TagTaskBatch, error) {
	var batches []model.TagTaskBatch
	err := model.DB.Order("id desc").Find(&batches).Error
	return batches, err
}

// ----------------- AI Engine API -----------------

func (a *App) ChatWithAI(message string) (string, error) {
	return a.aiEngine.ChatWithAI(a.ctx, message)
}

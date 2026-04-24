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

func (a *App) ImportData(filePath string) (int, error) {
	if filePath == "" {
		file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
			Title: "选择要导入的数据文件",
			Filters: []runtime.FileFilter{
				{DisplayName: "表格文件", Pattern: "*.csv;*.xlsx"},
			},
		})
		if err != nil {
			return 0, err
		}
		if file == "" {
			return 0, fmt.Errorf("cancelled")
		}
		filePath = file
	}
	return a.dataImport.ImportData(filePath)
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

func (a *App) GetRawDataList(page, pageSize int) (*PagedData, error) {
	var records []model.RawDataRecord
	var total int64

	db := model.DB.Model(&model.RawDataRecord{})
	db.Count(&total)

	offset := (page - 1) * pageSize
	err := db.Offset(offset).Limit(pageSize).Find(&records).Error
	if err != nil {
		return nil, err
	}

	return &PagedData{
		Total:   total,
		Records: records,
	}, nil
}

func (a *App) GetTaggedDataList(keyword, tag, batch string, page, pageSize int) (*model.PagedTaggedData, error) {
	// TODO: 完整的联表过滤查询逻辑
	// 当前先返回一个空结果，等联调 TaggedData 页面时再补充具体 SQL
	return &model.PagedTaggedData{
		Total:   0,
		Records: []model.TaggedRecordDto{},
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

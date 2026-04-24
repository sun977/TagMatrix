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
)

// App struct
type App struct {
	ctx          context.Context
	dataImport   *dataimport.DataImportService
	tagLogic     *taglogic.TagLogicService
	taskEngine   *taskengine.TaskEngineService
	aiEngine     *aiengine.AIEngineService
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

// ----------------- Data Import/Export API -----------------

func (a *App) ImportData(filePath string) (int, error) {
	return a.dataImport.ImportData(filePath)
}

func (a *App) ExportData(batchID uint64, exportPath string) error {
	return a.dataImport.ExportData(batchID, exportPath)
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

// ----------------- AI Engine API -----------------

func (a *App) ChatWithAI(message string) (string, error) {
	return a.aiEngine.ChatWithAI(a.ctx, message)
}

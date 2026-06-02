// ============================================================================
//  ___________              _____          __         .__
//  \__    ___/____    ____ /     \ _____ _/  |________|__|__  ___
//    |    |  \__  \  / ___/  \ /  \\__  \\   __\_  __ \  \  \/  /
//    |    |   / __ \/ /_/  >  Y    \/ __ \|  |  |  | \/  |>    <
//    |____|  (____  /\___  /____|__  (____  /__|  |__|  |__/__/\_ \
//                 \//_____/        \/     \/                     \/
// ============================================================================
// ⚡️ TagMatrix :: Wails Application Bridge
//
// 👤 SYSTEM_ARCHITECT : sun977 (SunHaobo)
// 🌐 GITHUB_REF       : https://github.com/sun977
// 📧 CONTACT_MAIL     : jiuwei977@foxmail.com
// 📅 INIT_YEAR        : 2026
//
// 📝 [DESC] 承载 Wails 前后端通信绑定的核心桥梁，串联 Vue3 UI 与底层 Go 业务服务的全局枢纽。
//
// 💡 "A somewhat obsessive developer in cybersecurity & AI scenarios."
// ============================================================================

package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	osRuntime "runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/pkg/logger"
	"TagMatrix/internal/service/aiengine"
	"TagMatrix/internal/service/dataadmin"
	"TagMatrix/internal/service/dataimport"
	"TagMatrix/internal/service/dataset"
	"TagMatrix/internal/service/taglogic"
	"TagMatrix/internal/service/taskengine"
	"TagMatrix/internal/service/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

//go:embed wails.json
var wailsJSON []byte

// 内嵌打包，从wails.json中读取相关的参数提供使用

// App struct
type App struct {
	ctx        context.Context
	dataset    *dataset.DatasetService
	dataImport *dataimport.DataImportService
	tagLogic   *taglogic.TagLogicService
	taskEngine *taskengine.TaskEngineService
	aiEngine   *aiengine.AIEngineService
	dataAdmin  *dataadmin.DataAdminService
	backupSvc  *dataadmin.BackupService

	dbPath  string
	logPath string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// 应用启动时调用startup函数。上下文已保存
// so we can call the runtime methods
// 因此我们可以调用运行时方法
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 确定数据存放目录
	var appDir string
	env := runtime.Environment(ctx)
	if env.BuildType == "dev" || env.BuildType == "debug" {
		// 开发模式：直接使用当前项目根目录
		appDir = "."
		// 不再用 fmt，等下面 logger 起来后再打，这里暂时留空或只保留极简输出
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
	a.dbPath, _ = filepath.Abs(dbPath)

	// 2. 初始化配置文件
	err := config.InitConfig(appDir)
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
	}

	logPath := filepath.Join(appDir, "app.log")
	a.logPath, _ = filepath.Abs(logPath)
	cfg := config.GetConfig()
	logger.InitLogger(logPath, cfg.Adv.DebugMode)
	logger.Info("TagMatrix application started", zap.String("appDir", appDir))

	// 1. 初始化数据库 (传入 logger)
	err = model.InitDB(dbPath)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// 在启动时，将配置文件中的 DebugMode 同步给 GORM Logger，否则默认是 Warn 级别
	model.UpdateDBLoggerLevel(cfg.Adv.DebugMode)

	// 3. 初始化所有的 Service
	a.dataset = dataset.NewDatasetService()
	a.dataImport = dataimport.NewDataImportService()
	a.tagLogic = taglogic.NewTagLogicService()
	a.taskEngine = taskengine.NewTaskEngineService(ctx)
	a.aiEngine = aiengine.NewAIEngineService()

	// Inject RunTaggingTaskFunc to break circular dependency
	a.aiEngine.RunTaggingTaskFunc = func(ctx context.Context, datasetID uint64, ruleIDs []uint64, tagMode string) (uint64, error) {
		// 生成带有时间戳和三位随机大写字母的任务名称，如 AITask_20260528_104223_QAZ
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		b := make([]byte, 3)
		for i := range b {
			b[i] = letters[r.Intn(len(letters))]
		}
		batchName := fmt.Sprintf("AITask_%s_%s", time.Now().Format("20060102_150405"), string(b))

		// 统一硬编码隐藏 AI 控制参数的考虑：
		// 1. 任务名称 (batchName) 与 任务描述 (desc)：固定统一带有 AITask 前缀，方便在前端任务看板中与人工发起任务进行区分追溯。
		// 2. 执行策略 (isOverwrite)：强制置为 false (追加模式)。防止 AI 幻觉或语义误解导致整个数据集已有标签被灾难性清空，确保数据安全。
		// 3. 来源文件列表 (sourceFiles)：强制传空数组 []string{}。代表对该数据集下全量数据执行，免去 AI 需额外调用工具查询文件名单的繁琐步骤，提升系统响应速度。
		return a.taskEngine.RunTaggingTask(datasetID, ruleIDs, batchName, "Created by AI Copilot", false, tagMode, []string{})
	}

	a.dataAdmin = dataadmin.NewDataAdminService(model.DB)
	a.backupSvc = dataadmin.NewBackupService(model.DB, appDir)

	// 获取 wails.json 中的应用版本信息
	var wailsConfig struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	currentVer := "v4.0.0" // default fallback
	if err := json.Unmarshal(wailsJSON, &wailsConfig); err == nil && wailsConfig.Info.ProductVersion != "" {
		currentVer = "v" + wailsConfig.Info.ProductVersion
	}

	// 异步检查更新 Check for updates asynchronously
	updater.CheckForUpdates(a.ctx, currentVer)
}

// ----------------- Config API -----------------

// GetAppConfig 获取当前应用的配置
func (a *App) GetAppConfig() config.AppConfig {
	return config.GetConfig()
}

// SaveAppConfig 保存应用的配置
func (a *App) SaveAppConfig(newConfig config.AppConfig) error {
	err := config.SaveConfig(newConfig)
	if err == nil {
		// 动态更新日志级别
		logger.SetDebugMode(newConfig.Adv.DebugMode)
		// 如果有必要，也可以将这个 DebugMode 同步到 GORM Logger，
		// 但目前我们已经在 logger.SetDebugMode 中处理了底层 Zap，
		// GORM 可能会自动跟随。稳妥起见，这里也可以通知 model 更新
		model.UpdateDBLoggerLevel(newConfig.Adv.DebugMode)
		// 这两行代码的结合，确保了从“数据产生源头（GORM）”到“最终写入端（Zap）”的日志级别是 完全同步、且性能最优的性能损耗最低的 。
	}
	return err
}

// BackupAppConfig 备份当前的配置文件
func (a *App) BackupAppConfig() (string, error) {
	return config.BackupConfig()
}

type AppPaths struct {
	DBPath  string `json:"dbPath"`
	LogPath string `json:"logPath"`
}

// GetAppPaths 获取应用存储路径
func (a *App) GetAppPaths() AppPaths {
	return AppPaths{
		DBPath:  a.dbPath,
		LogPath: a.logPath,
	}
}

// OpenDirectoryInOS 打开系统目录
func (a *App) OpenDirectoryInOS(path string) {
	dir := filepath.Dir(path)

	var cmd *exec.Cmd
	switch osRuntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", dir)
	}

	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to open directory: %v", err)
	}
}

// ----------------- Dashboard & Stats API -----------------

// GetDashboardStats 获取仪表盘统计信息
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

// GetDatasetTotalRecords 获取指定数据集的总记录数（轻量级接口）
func (a *App) GetDatasetTotalRecords(datasetID uint64) (int64, error) {
	return a.dataset.GetDatasetTotalRecords(datasetID)
}

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

// ExportDatasetWithRules 导出数据集和规则业务资产
func (a *App) ExportDatasetWithRules(datasetID uint64, exportPath string) error {
	if exportPath == "" {
		file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "导出业务资产",
			DefaultFilename: "dataset_with_rules.json",
			Filters: []runtime.FileFilter{
				{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
			},
		})
		if err != nil {
			return err
		}
		if file == "" {
			return nil // 用户取消
		}
		exportPath = file
	}
	return a.dataset.ExportDatasetWithRules(datasetID, exportPath)
}

// ImportDatasetWithRules 导入业务资产
func (a *App) ImportDatasetWithRules(filePath string) (*model.ImportResult, error) {
	if filePath == "" {
		file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
			Title: "导入业务资产",
			Filters: []runtime.FileFilter{
				{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
			},
		})
		if err != nil {
			return nil, err
		}
		if file == "" {
			return nil, nil // 用户取消
		}
		filePath = file
	}
	return a.dataset.ImportDatasetWithRules(filePath)
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

// ImportData 导入数据
func (a *App) ImportData(filePath string, selectedSheets []string, datasetID uint64, newDatasetName string) (int, error) {
	return a.dataImport.ImportData(filePath, selectedSheets, datasetID, newDatasetName)
}

// ExportData 导出数据
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

// GetRawDataList 获取原始数据列表
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

// DeleteRawData 删除原始数据
func (a *App) DeleteRawData(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	// GORM 软删除或硬删除，取决于 RawDataRecord 是否包含 gorm.DeletedAt
	return model.DB.Delete(&model.RawDataRecord{}, ids).Error
}

// GetTaggedDataList 获取打标数据列表
func (a *App) GetTaggedDataList(datasetID, keyword, tag, batch, searchCol, sourceFile, tagMode, status, startDate, endDate, isAiIntervened string, page, pageSize int) (*model.PagedTaggedData, error) {
	var total int64
	var dtos []model.TaggedRecordDto

	// 1. 构建查询构造器
	db := model.DB.Model(&model.RawDataRecord{})

	if datasetID != "" {
		db = db.Where("raw_data_records.dataset_id = ?", datasetID)
	}

	if keyword != "" {
		if searchCol != "" {
			// 根据指定列搜索
			db = db.Where("json_extract(raw_data_records.data, '$."+searchCol+"') LIKE ?", "%"+keyword+"%")
		} else {
			// 全局搜索
			db = db.Where("raw_data_records.data LIKE ?", "%"+keyword+"%")
		}
	}

	if sourceFile != "" {
		db = db.Where("json_extract(raw_data_records.data, '$.\"TagM_sourceFile\"') = ?", sourceFile)
	}

	if startDate != "" {
		db = db.Where("raw_data_records.updated_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		db = db.Where("raw_data_records.updated_at <= ?", endDate+" 23:59:59")
	}

	// 其他关联过滤条件需要在连接表上操作，因为我们需要分页
	// 由于 Tag, Batch, TagMode, Status 是多对多或根据计算得出的，我们最好使用 Join 或者子查询

	if tag != "" || batch != "" || tagMode != "" || status != "" || isAiIntervened != "" {
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

		switch isAiIntervened {
		case "true":
			subQuery = subQuery.Where("is_ai_intervened = ?", true)
		case "false":
			// 如果查未介入的，需要确保它的标签中没有一个是 is_ai_intervened = true 的
			// 所以先找到有哪些 record_id 是 true 的，然后排除它们，同时要确保它在 sys_entity_tags 里（排除掉未命中的，除非 status 选了全部）
			subQuery = subQuery.Where("sys_entity_tags.record_id NOT IN (?)", model.DB.Table("sys_entity_tags").Select("record_id").Where("is_ai_intervened = ?", true))
		}

		switch status {
		case "success":
			db = db.Where("raw_data_records.id IN (?)", subQuery)
		case "unmatched":
			db = db.Where("raw_data_records.id NOT IN (?)", subQuery)
		default:
			if isAiIntervened != "" {
				// 如果筛选了 AI 是否介入，隐含条件是它已经被打标了
				db = db.Where("raw_data_records.id IN (?)", subQuery)
			} else {
				db = db.Where("raw_data_records.id IN (?)", subQuery)
			}
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
			DatasetID:  r.DatasetID,
			Content:    r.Data, // 将原始数据内容传递给前端，前端可做解析
			UpdateTime: r.UpdatedAt.Format("2006-01-02 15:04:05"),
			Tags:       []model.TagDto{},
		}

		// 解析来源文件
		var dataMap map[string]interface{}
		if err := json.Unmarshal([]byte(r.Data), &dataMap); err == nil {
			if src, ok := dataMap["TagM_sourceFile"].(string); ok {
				dto.SourceFile = src
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

			// 构建 TagID 到 IsPrimary 的映射以及收集 TagHits
			primaryTagMap := make(map[uint64]bool)
			hitsMap := make(map[uint64]int)
			var tagIDs []uint64
			for _, et := range entityTags {
				tagIDs = append(tagIDs, et.TagID)
				if et.IsPrimary {
					primaryTagMap[et.TagID] = true
				}
				if et.Hits > 0 {
					hitsMap[et.TagID] = et.Hits
				}

				// 从 EntityTag 获取 MDCT 字段 (如果有多条，优先保留被AI介入过的数据或置信度最高的)
				if et.IsAiIntervened {
					dto.IsAiIntervened = true
					if dto.AiArbitrationReason == "" {
						dto.AiArbitrationReason = et.AiArbitrationReason
					}
				}
				if et.Confidence > dto.Confidence {
					dto.Confidence = et.Confidence
				}
			}

			if len(tagIDs) > 0 {
				var tags []model.SysTag
				model.DB.Where("id IN ?", tagIDs).Find(&tags)
				for _, t := range tags {
					displayName := t.Path
					if displayName == "" {
						displayName = t.Name
					}

					tagDto := model.TagDto{
						Name:  displayName,
						Color: t.Color,
					}
					dto.Tags = append(dto.Tags, tagDto)

					// 如果是主标签
					if primaryTagMap[t.ID] {
						dto.PrimaryTag = &tagDto
					}

					// 如果有命中次数，放入 DTO 的 TagHits 中
					if hitsMap[t.ID] > 0 {
						if dto.TagHits == nil {
							dto.TagHits = make(map[string]int)
						}
						dto.TagHits[displayName] = hitsMap[t.ID]
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
func (a *App) ExportTaggedDataList(datasetID, keyword, tag, batch, searchCol, sourceFile, tagMode, status, startDate, endDate, isAiIntervened string) error {
	// 构建查询条件
	db := model.DB.Model(&model.RawDataRecord{})

	if datasetID != "" {
		db = db.Where("raw_data_records.dataset_id = ?", datasetID)
	}

	if keyword != "" {
		if searchCol != "" {
			db = db.Where("json_extract(raw_data_records.data, '$."+searchCol+"') LIKE ?", "%"+keyword+"%")
		} else {
			db = db.Where("raw_data_records.data LIKE ?", "%"+keyword+"%")
		}
	}

	if sourceFile != "" {
		db = db.Where("json_extract(raw_data_records.data, '$.\"TagM_sourceFile\"') = ?", sourceFile)
	}

	if startDate != "" {
		db = db.Where("raw_data_records.updated_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		db = db.Where("raw_data_records.updated_at <= ?", endDate+" 23:59:59")
	}

	if tag != "" || batch != "" || tagMode != "" || status != "" || isAiIntervened != "" {
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

		switch isAiIntervened {
		case "true":
			subQuery = subQuery.Where("is_ai_intervened = ?", true)
		case "false":
			subQuery = subQuery.Where("sys_entity_tags.record_id NOT IN (?)", model.DB.Table("sys_entity_tags").Select("record_id").Where("is_ai_intervened = ?", true))
		}

		switch status {
		case "success":
			db = db.Where("raw_data_records.id IN (?)", subQuery)
		case "unmatched":
			db = db.Where("raw_data_records.id NOT IN (?)", subQuery)
		default:
			if isAiIntervened != "" {
				db = db.Where("raw_data_records.id IN (?)", subQuery)
			} else {
				db = db.Where("raw_data_records.id IN (?)", subQuery)
			}
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
		DefaultFilename: "tagged_data_export.xlsx",
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel 文件", Pattern: "*.xlsx"},
			{DisplayName: "CSV 文件", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return err
	}
	if filePath == "" {
		return fmt.Errorf("cancelled")
	}

	isExcel := strings.HasSuffix(strings.ToLower(filePath), ".xlsx")
	var (
		file   *os.File
		writer *csv.Writer
		f      *excelize.File
		sheet  string
		rowIdx int = 1
	)

	if isExcel {
		f = excelize.NewFile()
		sheet = "Sheet1"
		f.SetSheetName(f.GetSheetName(0), sheet)
	} else {
		var err error
		file, err = os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// 写入 UTF-8 BOM，防止 Excel 乱码
		file.Write([]byte("\xEF\xBB\xBF"))
		writer = csv.NewWriter(file)
		defer writer.Flush()
	}

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
			// 跳过 id 和 TagM_sourceFile
			if !colSet[k] && k != "id" && k != "TagM_sourceFile" {
				colSet[k] = true
				dynamicCols = append(dynamicCols, k)
			}
		}
	}

	sort.Strings(dynamicCols)
	// 构建表头 (不要系统 ID 和打标时间)
	headers := append([]string{}, dynamicCols...)
	headers = append(headers, "TagM_打标模式", "TagM_命中标签", "TagM_命中主标签", "TagM_副作用频次", "TagM_is_ai_intervened", "TagM_ai_arbitration_reason", "TagM_confidence", "TagM_任务批次", "TagM_sourceFile", "TagM_状态")

	if isExcel {
		for cIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowIdx)
			f.SetCellValue(sheet, cell, h)
		}
		rowIdx++
	} else {
		if err := writer.Write(headers); err != nil {
			return err
		}
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
		isAiIntervenedStr := "-"
		aiArbitrationReasonStr := "-"
		confidenceStr := "-"
		tagHitsStr := "-"

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

			// 确定谁是主标签并提取 MDCT 字段
			var primaryTagID uint64
			if tagModeVal == "single" && len(entityTags) > 0 {
				// 单标签模式下，唯一的标签就是主标签
				primaryTagID = entityTags[0].TagID
				primaryMap[primaryTagID] = true

				// 遍历找出 AI 介入过的标签，或者置信度最高的（这和列表页的显示逻辑保持一致）
				for _, et := range entityTags {
					if et.IsAiIntervened {
						isAiIntervenedStr = "true"
						aiArbitrationReasonStr = et.AiArbitrationReason
						confidenceStr = fmt.Sprintf("%.2f", et.Confidence)
						break
					}
					if et.Confidence > 0 && confidenceStr == "-" {
						isAiIntervenedStr = strconv.FormatBool(et.IsAiIntervened)
						aiArbitrationReasonStr = et.AiArbitrationReason
						confidenceStr = fmt.Sprintf("%.2f", et.Confidence)
					}
				}
				// 兜底：如果遍历完还没拿到，就取第一条的
				if confidenceStr == "-" {
					isAiIntervenedStr = strconv.FormatBool(entityTags[0].IsAiIntervened)
					aiArbitrationReasonStr = entityTags[0].AiArbitrationReason
					confidenceStr = fmt.Sprintf("%.2f", entityTags[0].Confidence)
				}
			} else {
				// 混合模式下，寻找 is_primary = true 的标签
				for _, et := range entityTags {
					if et.IsPrimary {
						primaryTagID = et.TagID
						primaryMap[primaryTagID] = true
						isAiIntervenedStr = strconv.FormatBool(et.IsAiIntervened)
						aiArbitrationReasonStr = et.AiArbitrationReason
						confidenceStr = fmt.Sprintf("%.2f", et.Confidence)
						break
					}
				}
			}

			for _, et := range entityTags {
				tagIDs = append(tagIDs, et.TagID)
			}

			if len(tagIDs) > 0 {
				var tags []model.SysTag
				model.DB.Where("id IN ?", tagIDs).Find(&tags)

				var tNames []string
				var hitsList []string
				for _, t := range tags {
					displayName := t.Path
					if displayName == "" {
						displayName = t.Name
					}

					tNames = append(tNames, displayName)
					if primaryMap[t.ID] {
						primaryTagStr = displayName
					}

					// 收集副作用计数
					for _, et := range entityTags {
						if et.TagID == t.ID && et.Hits > 0 {
							hitsList = append(hitsList, fmt.Sprintf("%s(%d次)", displayName, et.Hits))
							break
						}
					}
				}
				if len(tNames) > 0 {
					tagsStr = strings.Join(tNames, ", ")
				}
				if len(hitsList) > 0 {
					tagHitsStr = strings.Join(hitsList, ", ")
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

		// 来源文件
		sourceFileStr := "-"
		if ds, ok := dMap["TagM_sourceFile"].(string); ok {
			sourceFileStr = ds
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
		row = append(row, tagModeVal, tagsStr, primaryTagStr, tagHitsStr, isAiIntervenedStr, aiArbitrationReasonStr, confidenceStr, batchName, sourceFileStr, statusVal)

		if isExcel {
			for cIdx, val := range row {
				cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowIdx)
				f.SetCellValue(sheet, cell, val)
			}
			rowIdx++
		} else {
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	if isExcel {
		if err := f.SaveAs(filePath); err != nil {
			return err
		}
	}

	return nil
}

// ----------------- Tag & Rule Logic API -----------------

// CreateTag 创建标签
func (a *App) CreateTag(tag model.SysTag) error {
	return a.tagLogic.CreateTag(&tag)
}

// UpdateTag 更新标签
func (a *App) UpdateTag(tag model.SysTag) error {
	return a.tagLogic.UpdateTag(&tag)
}

// MoveTag 移动标签
func (a *App) MoveTag(tagID uint64, newParentID uint64) error {
	return a.tagLogic.MoveTag(tagID, newParentID)
}

// DeleteTag 删除标签
func (a *App) DeleteTag(id uint64) error {
	return a.tagLogic.DeleteTag(id)
}

// CheckTagHasRules 检查标签或其子标签是否配置了匹配规则
func (a *App) CheckTagHasRules(id uint64) (bool, error) {
	return a.tagLogic.CheckTagHasRules(id)
}

// GetAllTags 获取所有标签
func (a *App) GetAllTags() ([]model.SysTag, error) {
	return a.tagLogic.GetAllTags()
}

// GetTagTree 获取标签树
func (a *App) GetTagTree() ([]model.TagTreeNode, error) {
	return a.tagLogic.GetTagTree()
}

// GetTagByPath 根据路径获取标签
func (a *App) GetTagByPath(path string) (*model.SysTag, error) {
	return a.tagLogic.GetTagByPath(path)
}

// ExportTags 导出标签结构
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

// ImportTags 导入标签结构
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

// SaveRule 保存匹配规则
func (a *App) SaveRule(rule model.SysMatchRule) error {
	return a.tagLogic.SaveRule(&rule)
}

// GetRulesByTag 获取某个标签下的所有规则
func (a *App) GetRulesByTag(tagID uint64) ([]model.SysMatchRule, error) {
	return a.tagLogic.GetRulesByTagID(tagID)
}

// GetRulesByDataset 获取某个数据集下的所有规则 (打标任务引擎使用的批量拉取接口)
func (a *App) GetRulesByDataset(datasetID uint64) ([]model.SysMatchRule, error) {
	return a.tagLogic.GetRulesByDataset(datasetID)
}

// DeleteRule 删除规则
func (a *App) DeleteRule(id uint64) error {
	return a.tagLogic.DeleteRule(id)
}

// GetAllRules 获取所有规则
func (a *App) GetAllRules() ([]model.SysMatchRule, error) {
	return a.tagLogic.GetAllRules()
}

// DryRunRule 规则试算
func (a *App) DryRunRule(ruleJSON string, limit int, datasetID uint64) ([]taglogic.DryRunResult, error) {
	return a.tagLogic.DryRunRule(ruleJSON, limit, datasetID)
}

// CloneRule 克隆规则
func (a *App) CloneRule(sourceRuleID uint64, targetDatasetID uint64, tagID uint64) (*taglogic.CloneRuleResult, error) {
	return a.tagLogic.CloneRule(sourceRuleID, targetDatasetID, tagID)
}

// InheritRules 批量继承规则
func (a *App) InheritRules(sourceDatasetID uint64, targetDatasetID uint64, ruleIDs []uint64) (*taglogic.InheritRulesResult, error) {
	return a.tagLogic.InheritRules(sourceDatasetID, targetDatasetID, ruleIDs)
}

// ----------------- Task Engine API -----------------
// RunTaggingTask 异步执行规则打标任务
func (a *App) RunTaggingTask(datasetID uint64, ruleIDs []uint64, batchName string, desc string, isOverwrite bool, tagMode string, sourceFiles []string) (uint64, error) {
	return a.taskEngine.RunTaggingTask(datasetID, ruleIDs, batchName, desc, isOverwrite, tagMode, sourceFiles)
}

// GetAvailableSourceFiles 获取所有可用的来源文件选项
func (a *App) GetAvailableSourceFiles(datasetID uint64) ([]model.SourceFileOption, error) {
	return a.taskEngine.GetAvailableSourceFiles(a.ctx, datasetID)
}

// RollbackTask 回退指定的打标批次
func (a *App) RollbackTask(batchID uint64) error {
	return a.taskEngine.RollbackTask(a.ctx, batchID)
}

// DeleteTaskBatches 硬删除指定的打标批次及其关联的日志和标签
func (a *App) DeleteTaskBatches(batchIDs []uint64) error {
	return a.taskEngine.DeleteTaskBatches(a.ctx, batchIDs)
}

// GetTaskBatches 获取所有打标批次
func (a *App) GetTaskBatches() ([]model.TagTaskBatch, error) {
	return a.taskEngine.GetTaskBatches()
}

// GetTaskLogs 获取某个批次的打标日志
func (a *App) GetTaskLogs(batchID uint64) ([]model.TagTaskLogDto, error) {
	return a.taskEngine.GetTaskLogs(batchID)
}

// GetTaskLogsPaged 分页获取某个批次的打标日志
func (a *App) GetTaskLogsPaged(batchID uint64, page int, pageSize int) (*model.PagedTaskLogs, error) {
	return a.taskEngine.GetTaskLogsPaged(batchID, page, pageSize)
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

// SaveCSVFile saves a raw CSV string to a file using native dialog
// 选择保存路径
func (a *App) SaveCSVFile(defaultFilename string, csvContent string) (string, error) {
	filepath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Title:           "保存 CSV 文件",
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
		return "", nil // User cancelled
	}

	err = os.WriteFile(filepath, []byte(csvContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}
	return filepath, nil
}

// ----------------- AI Engine API -----------------

// 聊天
func (a *App) ChatWithAI(message string) (string, error) {
	return a.aiEngine.ChatWithAI(a.ctx, message)
}

// 聊天流式传输
func (a *App) ChatWithAIStream(reqId string, message string) error {
	return a.aiEngine.ChatWithAIStream(a.ctx, reqId, message)
}

// 测试 AI 连接
func (a *App) TestAIConnection(apiKey, baseUrl, modelName string) error {
	return a.aiEngine.TestConnection(a.ctx, apiKey, baseUrl, modelName)
}

// CheckUpdateManual 手动触发检查更新
func (a *App) CheckUpdateManual() (*updater.UpdateInfo, error) {
	// 获取 wails.json 中的应用版本信息
	var wailsConfig struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	currentVer := "v4.0.0"
	if err := json.Unmarshal(wailsJSON, &wailsConfig); err == nil && wailsConfig.Info.ProductVersion != "" {
		currentVer = "v" + wailsConfig.Info.ProductVersion
	}

	info, err := updater.FetchLatestRelease(currentVer)
	if err != nil {
		logger.Log.Warn("Manual check update failed", zap.Error(err))
		return nil, err
	}
	return info, nil
}

// ----------------- DataAdmin API -----------------

// 执行原始 SQL
func (a *App) ExecuteRawSQL(query string) (*dataadmin.RawSQLResult, error) {
	return a.dataAdmin.ExecuteRawSQL(query)
}

// 获取系统表
func (a *App) GetSystemTables() ([]string, error) {
	return a.dataAdmin.GetSystemTables()
}

// 获取表数据
func (a *App) GetTableData(tableName string, offset, limit int) (*dataadmin.PagedTableData, error) {
	return a.dataAdmin.GetTableData(tableName, offset, limit)
}

// 获取虚拟数据集数据
func (a *App) GetVirtualDatasetData(datasetId uint, offset, limit int) (*dataadmin.PagedTableData, error) {
	return a.dataAdmin.GetVirtualDatasetData(datasetId, offset, limit)
}

// 插入系统表记录
func (a *App) InsertSystemTableRecord(tableName string, payload map[string]interface{}) error {
	return a.dataAdmin.InsertSystemTableRecord(tableName, payload)
}

// 更新系统表记录
func (a *App) UpdateSystemTableRecord(tableName string, recordId interface{}, payload map[string]interface{}) error {
	return a.dataAdmin.UpdateSystemTableRecord(tableName, recordId, payload)
}

// 删除系统表记录
func (a *App) DeleteSystemTableRecord(tableName string, recordId interface{}) error {
	return a.dataAdmin.DeleteSystemTableRecord(tableName, recordId)
}

// 插入虚拟数据记录
func (a *App) InsertVirtualRecord(datasetId uint, payload map[string]interface{}) error {
	return a.dataAdmin.InsertVirtualRecord(datasetId, payload)
}

// 更新虚拟数据记录
func (a *App) UpdateVirtualRecord(recordId uint, payload map[string]interface{}) error {
	return a.dataAdmin.UpdateVirtualRecord(recordId, payload)
}

// 删除虚拟数据记录
func (a *App) DeleteVirtualRecord(recordId uint) error {
	return a.dataAdmin.DeleteVirtualRecord(recordId)
}

// 获取 SQL 模板
func (a *App) GetSqlTemplates() ([]model.SysSqlTemplate, error) {
	return a.dataAdmin.GetSqlTemplates()
}

// 保存 SQl 模板
func (a *App) SaveSqlTemplate(id uint64, name, query string) error {
	return a.dataAdmin.SaveSqlTemplate(id, name, query)
}

// 删除 SQL 模板
func (a *App) DeleteSqlTemplate(id uint64) error {
	return a.dataAdmin.DeleteSqlTemplate(id)
}

// ----------------- Backup Service API -----------------
// 列出备份清单
func (a *App) ListBackups() ([]dataadmin.BackupInfo, error) {
	return a.backupSvc.ListBackups()
}

// 创建备份
func (a *App) CreateBackup(note string) error {
	return a.backupSvc.CreateBackup(note)
}

// 删除备份
func (a *App) DeleteBackup(backupPath string) error {
	return a.backupSvc.DeleteBackup(backupPath)
}

// 重载数据库
func (a *App) RestoreDatabase(backupPath string) error {
	// 1. 关闭现有数据库连接释放文件锁
	sqlDB, err := model.DB.DB()
	if err == nil {
		sqlDB.Close()
	}

	// 2. 覆盖数据库文件
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %v", err)
	}
	defer src.Close()

	// 确定数据存放目录
	var appDir string
	env := runtime.Environment(a.ctx)
	if env.BuildType == "dev" || env.BuildType == "debug" {
		appDir = "."
	} else {
		appDataDir, _ := os.UserConfigDir()
		appDir = filepath.Join(appDataDir, "TagMatrix")
	}
	dbPath := filepath.Join(appDir, "data.db")

	dest, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to create target db file: %v", err)
	}

	_, err = io.Copy(dest, src)
	dest.Close() // 复制完立即关闭

	if err != nil {
		return fmt.Errorf("failed to restore database file: %v", err)
	}

	// 3. 重新初始化所有服务和连接
	a.startup(a.ctx)

	return nil
}

// 导入数据库文件
func (a *App) ImportExternalDatabase() error {
	filepath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要导入的数据库文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite 数据库 (*.db)", Pattern: "*.db"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to open file dialog: %v", err)
	}
	if filepath == "" {
		return fmt.Errorf("cancelled")
	}

	return a.backupSvc.RestoreDatabase(filepath)
}

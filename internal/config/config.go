package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// AppConfig 定义了整个应用的配置结构
type AppConfig struct {
	AI     AIConfig     `json:"ai"`
	System SystemConfig `json:"system"`
	Adv    AdvConfig    `json:"adv"`
}

// AIConfig 定义了 AI 相关的配置
type AIConfig struct {
	APIKey       string  `json:"api_key"`
	BaseURL      string  `json:"base_url"`
	Model        string  `json:"model"`
	Temperature  float64 `json:"temperature"`
	SystemPrompt string  `json:"system_prompt"`
}

// SystemConfig 定义了系统相关的配置
type SystemConfig struct {
	DefaultMode      string `json:"default_mode"`
	AutoBackup       bool   `json:"auto_backup"`
	TaskNotification bool   `json:"task_notification"`
	PreviewRows      int    `json:"preview_rows"`
}

// AdvConfig 定义了高级配置
type AdvConfig struct {
	Concurrency   int  `json:"concurrency"`
	Retries       int  `json:"retries"`
	DebugMode     bool `json:"debug_mode"`
	DeveloperMode bool `json:"developer_mode"`	// 开发者模式
}

var (
	configInstance *AppConfig
	configPath     string
	mu             sync.RWMutex
)

// InitConfig 初始化配置，如果配置文件不存在则生成默认配置
func InitConfig(appDataDir string) error {
	mu.Lock()
	defer mu.Unlock()

	configPath = filepath.Join(appDataDir, "config.json")

	// 如果文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &AppConfig{
			AI: AIConfig{
				APIKey:      "",
				BaseURL:     "https://api.openai.com/v1",
				Model:       "gpt-4o-mini",
				Temperature: 0.7,
				SystemPrompt: `你是一个专业的数据分析和打标辅助助手。
你的主要任务是：
1. 解答用户关于 TagMatrix 数据打标系统如何使用的问题。
2. 根据用户提供的需求，生成针对当前 SQLite 数据库的准确查询 SQL。
3. 帮助用户分析数据特征。

请注意：
- 用户的原始导入数据存储在 raw_data_records 表的 data 字段中（JSON 格式）。在 SQLite 中查询 JSON 数据请使用 json_extract 函数。
- 给出 SQL 时请使用 markdown 代码块包裹，以便前端渲染。`,
			},
			System: SystemConfig{
				DefaultMode:      "overwrite",
				AutoBackup:       true,
				TaskNotification: true,
				PreviewRows:      20,
			},
			Adv: AdvConfig{
				Concurrency: 5,
				Retries:     3,
				DebugMode:   false,
			},
		}

		if err := saveConfigToFile(defaultConfig, configPath); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
		configInstance = defaultConfig
		return nil
	}

	// 如果文件存在，读取并解析
	return loadConfigFromFile()
}

// GetConfig 获取当前配置的深拷贝，避免外部直接修改
func GetConfig() AppConfig {
	mu.RLock()
	defer mu.RUnlock()

	if configInstance == nil {
		return AppConfig{}
	}
	return *configInstance
}

// SaveConfig 保存新的配置到文件并更新内存中的实例
func SaveConfig(newConfig AppConfig) error {
	mu.Lock()
	defer mu.Unlock()

	if configPath == "" {
		return fmt.Errorf("config not initialized")
	}

	if err := saveConfigToFile(&newConfig, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configInstance = &newConfig
	return nil
}

// 内部方法：将配置序列化写入文件
func saveConfigToFile(cfg *AppConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// 内部方法：从文件加载配置
func loadConfigFromFile() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config json: %w", err)
	}

	configInstance = &cfg
	return nil
}

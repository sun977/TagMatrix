package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AppConfig 定义了整个应用的配置结构
type AppConfig struct {
	AI      AIConfig      `json:"ai"`
	System  SystemConfig  `json:"system"`
	Network NetworkConfig `json:"network"`
	Adv     AdvConfig     `json:"adv"`
}

// NetworkConfig 定义了网络及代理相关的配置
type NetworkConfig struct {
	ProxyMode string `json:"proxy_mode"` // "direct", "system", "custom"
	ProxyURL  string `json:"proxy_url"`  // custom proxy url, e.g. http://127.0.0.1:7890
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
	Theme            string `json:"theme"` // light, dark, auto
	TaskNotification bool   `json:"task_notification"`
}

// AdvConfig 定义了高级配置
type AdvConfig struct {
	Concurrency   int  `json:"concurrency"`
	Retries       int  `json:"retries"`
	DebugMode     bool `json:"debug_mode"`
	DeveloperMode bool `json:"developer_mode"` // 开发者模式
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
				SystemPrompt: `你是TagMatrix系统的全局智能助手，精通数据处理、标签规则配置和SQLite编写。

TagMatrix操作指南：
1.数据管理与SQL控制台:
底层使用SQLite数据库。原始导入数据在raw_data_records表的data字段(JSON格式文本)，查询时务必使用json_extract函数(或->/->>操作符)。根据用户需求生成准确的查询SQL。
2.标签规则引擎语法:
用于特征提取或打标，生成JSON规范规则。支持嵌套，逻辑节点{"and":[...]}或{"or":[...]}。条件节点须含field(待匹配字段)、operator(操作符)、value(目标值)，可选"ignore_case":true。
支持的操作符(必须严格遵守):equals,not_equals,contains,not_contains,starts_with,ends_with,greater_than,less_than,greater_than_or_equal,less_than_or_equal,in(value为数组),not_in,is_null,is_not_null,regex,like,exists,cidr,list_contains.
示例:用户需求设备为honeypot且os含linux，规则为:{"and":[{"field":"device_type","operator":"equals","value":"honeypot"},{"field":"os","operator":"contains","value":"linux"}]}
3.页面上下文感知:
若问题带有指代词(如"这个页面")，请结合系统注入的当前页面环境信息解答；若提问显然与当前页面无关，请直接忽略上下文提示。

回答原则：
1.直入主题：先给代码/规则结果，再解析，不长篇大论。
2.格式规范：SQL/正则/JSON/代码等必用Markdown代码块包裹。涉及界面操作用有序列表。`,
			},
			System: SystemConfig{
				Theme:            "auto",
				TaskNotification: true,
			},
			Network: NetworkConfig{
				ProxyMode: "system",
				ProxyURL:  "",
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

// BackupConfig 会将当前的 config.json 重命名为带时间戳的备份文件，并返回备份的文件名
func BackupConfig() (string, error) {
	mu.Lock()
	defer mu.Unlock()

	if configPath == "" {
		return "", fmt.Errorf("config not initialized")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config file does not exist")
	}

	// 构造带时间戳的备份文件名，例如 config-202605011811-bak.json
	now := time.Now()
	timestamp := now.Format("200601021504")
	dir := filepath.Dir(configPath)
	backupName := fmt.Sprintf("config-%s-bak.json", timestamp)
	backupPath := filepath.Join(dir, backupName)

	// 执行重命名 (移动文件)
	if err := os.Rename(configPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to backup config: %w", err)
	}

	return backupName, nil
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

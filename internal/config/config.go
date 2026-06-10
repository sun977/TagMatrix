package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AppConfig 定义了整个应用的配置结构
type AppConfig struct {
	AI      AIConfig      `json:"ai"`
	System  SystemConfig  `json:"system"`
	Network NetworkConfig `json:"network"`
	Adv     AdvConfig     `json:"adv"`
	MDCT    MDCTConfig    `json:"mdct"`
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
	SystemPrompt string  `json:"system_prompt"` // 废弃，保留用于兼容老版本【系统的prompt由程序工程化管理，不再需要用户设置这个字段】
	CustomPrompt string  `json:"custom_prompt"`
}

// SystemConfig 定义了系统相关的配置
type SystemConfig struct {
	Theme            string `json:"theme"` // light, dark, auto
	TaskNotification bool   `json:"task_notification"`
}

// AdvConfig 定义了高级配置
type AdvConfig struct {
	Concurrency     int  `json:"concurrency"`      // AI 请求并发数
	TaskConcurrency int  `json:"task_concurrency"` // 本地任务处理并发数
	Retries         int  `json:"retries"`          // AI 请求失败重试次数
	DebugMode       bool `json:"debug_mode"`       // DUBUG 模式
	DeveloperMode   bool `json:"developer_mode"`   // 开发者模式
}

// MDCTConfig 多维共识打标算法权重配置
type MDCTConfig struct {
	W1             int  `json:"w1"`               // 人为静态权重 (基础决定权)
	W2             int  `json:"w2"`               // 规则逻辑深度打分权重
	W3             int  `json:"w3"`               // 数据置信度打分权重
	W4             int  `json:"w4"`               // AI 语义裁决分权重
	AllowAiArbiter bool `json:"allow_ai_arbiter"` // 是否允许AI介入裁决 (对应文档中的数据隐私手动开关)
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
				APIKey:       "",
				BaseURL:      "https://api.openai.com/v1",
				Model:        "gpt-4o-mini",
				Temperature:  0.7,
				SystemPrompt: "",
				CustomPrompt: "",
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
				Concurrency:     5,
				TaskConcurrency: 20,
				Retries:         1,
				DebugMode:       false,
			},
			MDCT: MDCTConfig{
				W1:             1000,
				W2:             10,
				W3:             10,
				W4:             100,
				AllowAiArbiter: false, // 默认关闭，保护数据隐私
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

	// 兼容老版本配置文件：如果 MDCT 权重全是 0，说明是旧配置升级上来的，补全推荐默认值
	if cfg.MDCT.W1 == 0 && cfg.MDCT.W2 == 0 && cfg.MDCT.W3 == 0 && cfg.MDCT.W4 == 0 {
		cfg.MDCT.W1 = 1000
		cfg.MDCT.W2 = 10
		cfg.MDCT.W3 = 10
		cfg.MDCT.W4 = 100
	}

	// 兼容 SystemPrompt 平滑迁移到 CustomPrompt
	if cfg.AI.SystemPrompt != "" {
		if strings.Contains(cfg.AI.SystemPrompt, "你是TagMatrix系统的全局智能助手") {
			// 如果包含旧版默认的特征词，说明用户没有修改过，直接丢弃
			cfg.AI.SystemPrompt = ""
		} else {
			// 如果是用户自写的业务内容，将其迁移至 CustomPrompt
			if cfg.AI.CustomPrompt == "" {
				cfg.AI.CustomPrompt = cfg.AI.SystemPrompt
			}
			cfg.AI.SystemPrompt = ""
		}

		// 同步写回文件确保配置瘦身 (静默执行，失败不影响主流程)
		_ = saveConfigToFile(&cfg, configPath)
	}

	configInstance = &cfg
	return nil
}

package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型，提供统一的主键和时间追踪字段
type BaseModel struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"` // 数据库字段类型为 bigint unsigned ，对应 Go 语言 uint64
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}

// SysDataset 数据集管理表
type SysDataset struct {
	BaseModel
	Name        string `json:"name" gorm:"size:100;not null;uniqueIndex;comment:数据集名称"`
	Description string `json:"description" gorm:"size:255;comment:描述"`
	SchemaKeys  string `json:"schema_keys" gorm:"type:text;comment:JSON格式的表头字段数组"`
}

// RawDataRecord 原始数据表，用于动态存储导入的 Excel/CSV 数据
type RawDataRecord struct {
	BaseModel
	DatasetID uint64         `json:"dataset_id" gorm:"index;not null;comment:关联的数据集ID"`
	BatchID   uint64         `json:"batch_id" gorm:"index;comment:导入时的批次 ID"`
	Data      string         `json:"data" gorm:"type:text;comment:动态列数据 (建议存储 JSON 字符串)"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// SysTag 标签定义表 (树状结构)
type SysTag struct {
	BaseModel
	Name        string         `json:"name" gorm:"size:100;not null;comment:标签名称"`
	ParentID    uint64         `json:"parent_id" gorm:"default:0;comment:父级标签ID，0表示根节点"`
	Path        string         `json:"path" gorm:"size:700;index;comment:物理路径 (Materialized Path) e.g. /1/5/10/"`
	Level       int            `json:"level" gorm:"default:0;comment:层级深度"`
	Color       string         `json:"color" gorm:"size:7;comment:标签颜色 (HEX)"`
	Description string         `json:"description" gorm:"size:255;comment:标签描述说明"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// SysMatchRule 自动打标规则表
type SysMatchRule struct {
	BaseModel
	TagID     uint64         `json:"tag_id" gorm:"index;not null;comment:关联的标签ID"`
	Name      string         `json:"name" gorm:"size:100;comment:规则名称"`
	Priority  int            `json:"priority" gorm:"default:0;comment:优先级 (越大越先匹配)"`
	RuleJSON  string         `json:"rule_json" gorm:"type:text;not null;comment:JSON格式的匹配规则 (matcher.MatchRule)"`
	IsEnabled bool           `json:"is_enabled" gorm:"default:true;index;comment:是否启用"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// TagTaskBatch 打标任务批次表，用于记录打标版本和回退
type TagTaskBatch struct {
	BaseModel
	Name           string         `json:"name" gorm:"size:100;comment:任务名称，比如 20240101-打标任务"`
	Status         string         `json:"status" gorm:"size:20;index;comment:状态：running, completed, failed, rolled_back"`
	TotalProcessed int            `json:"total_processed" gorm:"default:0;comment:总处理条数"`
	TagMode        string         `json:"tag_mode" gorm:"size:20;comment:打标模式：single, multiple, mixed"`
	DataSource     string         `json:"data_source" gorm:"size:255;comment:数据来源过滤条件"`
	FinishedAt     *time.Time     `json:"finished_at" gorm:"comment:任务完成时间"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index;comment:软删除时间"`
}

// TagTaskLog 打标操作审计日志，极详尽的记录，也是回退的依据
type TagTaskLog struct {
	BaseModel
	BatchID  uint64 `json:"batch_id" gorm:"index;not null;comment:关联任务批次ID"`
	RecordID uint64 `json:"record_id" gorm:"index;not null;comment:关联原始数据ID"`
	TagID    uint64 `json:"tag_id" gorm:"index;not null;comment:打上的标签ID"`
	RuleID   uint64 `json:"rule_id" gorm:"default:0;comment:命中的规则ID"`
	Reason   string `json:"reason" gorm:"type:text;comment:详细匹配过程/AI解释"`
	Action   string `json:"action" gorm:"size:20;comment:操作类型：add, remove"`
}

// SysEntityTag 实体-标签关联表 (最终的打标结果)
type SysEntityTag struct {
	BaseModel
	RecordID  uint64 `json:"record_id" gorm:"index:idx_record_tag;not null;comment:关联原始数据ID"`
	TagID     uint64 `json:"tag_id" gorm:"index:idx_record_tag;not null;comment:关联标签ID"`
	Source    string `json:"source" gorm:"size:50;default:'manual';comment:来源：manual, auto_rule, ai"`
	IsPrimary bool   `json:"is_primary" gorm:"default:false;comment:是否为主标签 (区分混合模式主副标签)"`
	BatchID   uint64 `json:"batch_id" gorm:"index;comment:记录是哪个批次打上的，用于回退"`
	RuleID    uint64 `json:"rule_id" gorm:"default:0;comment:如果是 auto_rule，记录命中的规则ID"`
}

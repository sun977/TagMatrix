package model

// DashboardStats 用于概览控制台的数据统计
type DashboardStats struct {
	TotalRecords  int64         `json:"totalRecords"`
	TaggedRecords int64         `json:"taggedRecords"`
	TotalTags     int64         `json:"totalTags"`
	TotalRules    int64         `json:"totalRules"`
	DatasetStats  []DatasetStat `json:"datasetStats"`
}

// DatasetStat 数据集级别的统计信息
type DatasetStat struct {
	DatasetID     uint64 `json:"datasetId"`
	DatasetName   string `json:"datasetName"`
	TotalRecords  int64  `json:"totalRecords"`
	TaggedRecords int64  `json:"taggedRecords"`
}

// TaggedRecordDto 用于展示打标结果的 DTO
type TaggedRecordDto struct {
	ID         uint64   `json:"id"`
	Content    string   `json:"content"`
	Tags       []TagDto `json:"tags"`
	PrimaryTag *TagDto  `json:"primaryTag,omitempty"` // 主标签（混合模式下）
	BatchName  string   `json:"batchName"`
	TagMode    string   `json:"tagMode"`    // 打标模式：multiple, single, mixed
	DataSource string   `json:"dataSource"` // 数据来源
	UpdateTime string   `json:"updateTime"`
	Status     string   `json:"status"` // success 或 unmatched
}

type TagDto struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// PagedTaggedData 分页打标结果
type PagedTaggedData struct {
	Total   int64             `json:"total"`
	Records []TaggedRecordDto `json:"records"`
}

// FileAnalysisResult 用于文件分析后的返回结果
type FileAnalysisResult struct {
	FilePath   string   `json:"filePath"`
	FileName   string   `json:"fileName"`
	FileType   string   `json:"fileType"`   // csv 或 excel
	SheetNames []string `json:"sheetNames"` // 只有 excel 有
}

// TagTreeNode 用于前端标签树组件展示
type TagTreeNode struct {
	SysTag
	HasRule  bool          `json:"has_rule"` // 是否配置了匹配规则
	Children []TagTreeNode `json:"children"`
}

// TagTaskLogDto 打标日志前端展示
type TagTaskLogDto struct {
	ID        uint64 `json:"id"`
	RecordID  uint64 `json:"recordId"`
	TagName   string `json:"tagName"`
	RuleName  string `json:"ruleName"`
	Action    string `json:"action"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"createdAt"`
}

// ExportTagNode 用于导出为 JSON 的精简结构，去除了数据库 ID 和时间戳等无用信息
type ExportTagNode struct {
	Name        string          `json:"name"`
	ParentID    uint64          `json:"parent_id"`
	Path        string          `json:"path"`
	Level       int             `json:"level"`
	Color       string          `json:"color"`
	Description string          `json:"description"`
	Children    []ExportTagNode `json:"children,omitempty"`
}

// DataSourceOption 打标任务数据源选项
type DataSourceOption struct {
	SourceName string `json:"source_name" gorm:"column:source_name"`
	Count      int64  `json:"count" gorm:"column:count"`
}

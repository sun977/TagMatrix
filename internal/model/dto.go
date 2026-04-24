package model

// DashboardStats 用于概览控制台的数据统计
type DashboardStats struct {
	TotalRecords  int64 `json:"totalRecords"`
	TaggedRecords int64 `json:"taggedRecords"`
	TotalTags     int64 `json:"totalTags"`
	TotalRules    int64 `json:"totalRules"`
}

// TaggedRecordDto 用于展示打标结果的 DTO
type TaggedRecordDto struct {
	ID         uint64   `json:"id"`
	Content    string   `json:"content"`
	Tags       []TagDto `json:"tags"`
	BatchName  string   `json:"batchName"`
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

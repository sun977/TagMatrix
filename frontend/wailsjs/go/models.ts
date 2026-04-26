export namespace config {
	
	export class AIConfig {
	    api_key: string;
	    base_url: string;
	    model: string;
	    temperature: number;
	    system_prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new AIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	        this.model = source["model"];
	        this.temperature = source["temperature"];
	        this.system_prompt = source["system_prompt"];
	    }
	}
	export class AdvConfig {
	    concurrency: number;
	    retries: number;
	    debug_mode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AdvConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.concurrency = source["concurrency"];
	        this.retries = source["retries"];
	        this.debug_mode = source["debug_mode"];
	    }
	}
	export class SystemConfig {
	    default_mode: string;
	    auto_backup: boolean;
	    task_notification: boolean;
	    preview_rows: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.default_mode = source["default_mode"];
	        this.auto_backup = source["auto_backup"];
	        this.task_notification = source["task_notification"];
	        this.preview_rows = source["preview_rows"];
	    }
	}
	export class AppConfig {
	    ai: AIConfig;
	    system: SystemConfig;
	    adv: AdvConfig;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ai = this.convertValues(source["ai"], AIConfig);
	        this.system = this.convertValues(source["system"], SystemConfig);
	        this.adv = this.convertValues(source["adv"], AdvConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace gorm {
	
	export class DeletedAt {
	    // Go type: time
	    Time: any;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeletedAt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = this.convertValues(source["Time"], null);
	        this.Valid = source["Valid"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class PagedData {
	    Total: number;
	    Records: model.RawDataRecord[];
	
	    static createFrom(source: any = {}) {
	        return new PagedData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Total = source["Total"];
	        this.Records = this.convertValues(source["Records"], model.RawDataRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace model {
	
	export class DashboardStats {
	    totalRecords: number;
	    taggedRecords: number;
	    totalTags: number;
	    totalRules: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRecords = source["totalRecords"];
	        this.taggedRecords = source["taggedRecords"];
	        this.totalTags = source["totalTags"];
	        this.totalRules = source["totalRules"];
	    }
	}
	export class DataSourceOption {
	    source_name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DataSourceOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_name = source["source_name"];
	        this.count = source["count"];
	    }
	}
	export class FileAnalysisResult {
	    filePath: string;
	    fileName: string;
	    fileType: string;
	    sheetNames: string[];
	
	    static createFrom(source: any = {}) {
	        return new FileAnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.fileType = source["fileType"];
	        this.sheetNames = source["sheetNames"];
	    }
	}
	export class TagDto {
	    name: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new TagDto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	export class TaggedRecordDto {
	    id: number;
	    content: string;
	    tags: TagDto[];
	    primaryTag?: TagDto;
	    batchName: string;
	    tagMode: string;
	    dataSource: string;
	    updateTime: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new TaggedRecordDto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.tags = this.convertValues(source["tags"], TagDto);
	        this.primaryTag = this.convertValues(source["primaryTag"], TagDto);
	        this.batchName = source["batchName"];
	        this.tagMode = source["tagMode"];
	        this.dataSource = source["dataSource"];
	        this.updateTime = source["updateTime"];
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PagedTaggedData {
	    total: number;
	    records: TaggedRecordDto[];
	
	    static createFrom(source: any = {}) {
	        return new PagedTaggedData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.records = this.convertValues(source["records"], TaggedRecordDto);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RawDataRecord {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    dataset_id: number;
	    batch_id: number;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new RawDataRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.dataset_id = source["dataset_id"];
	        this.batch_id = source["batch_id"];
	        this.data = source["data"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SysDataset {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    name: string;
	    description: string;
	    schema_keys: string;
	
	    static createFrom(source: any = {}) {
	        return new SysDataset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.schema_keys = source["schema_keys"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SysMatchRule {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    dataset_id: number;
	    tag_id: number;
	    name: string;
	    priority: number;
	    rule_json: string;
	    is_enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SysMatchRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.dataset_id = source["dataset_id"];
	        this.tag_id = source["tag_id"];
	        this.name = source["name"];
	        this.priority = source["priority"];
	        this.rule_json = source["rule_json"];
	        this.is_enabled = source["is_enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SysTag {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    name: string;
	    parent_id: number;
	    path: string;
	    level: number;
	    color: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new SysTag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.name = source["name"];
	        this.parent_id = source["parent_id"];
	        this.path = source["path"];
	        this.level = source["level"];
	        this.color = source["color"];
	        this.description = source["description"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TagTaskBatch {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    dataset_id: number;
	    name: string;
	    status: string;
	    total_processed: number;
	    tag_mode: string;
	    data_source: string;
	    // Go type: time
	    finished_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new TagTaskBatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.dataset_id = source["dataset_id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.total_processed = source["total_processed"];
	        this.tag_mode = source["tag_mode"];
	        this.data_source = source["data_source"];
	        this.finished_at = this.convertValues(source["finished_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TagTaskLogDto {
	    id: number;
	    recordId: number;
	    tagName: string;
	    ruleName: string;
	    action: string;
	    reason: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new TagTaskLogDto(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.recordId = source["recordId"];
	        this.tagName = source["tagName"];
	        this.ruleName = source["ruleName"];
	        this.action = source["action"];
	        this.reason = source["reason"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class TagTreeNode {
	    id: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    name: string;
	    parent_id: number;
	    path: string;
	    level: number;
	    color: string;
	    description: string;
	    has_rule: boolean;
	    children: TagTreeNode[];
	
	    static createFrom(source: any = {}) {
	        return new TagTreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.name = source["name"];
	        this.parent_id = source["parent_id"];
	        this.path = source["path"];
	        this.level = source["level"];
	        this.color = source["color"];
	        this.description = source["description"];
	        this.has_rule = source["has_rule"];
	        this.children = this.convertValues(source["children"], TagTreeNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace taglogic {
	
	export class DryRunResult {
	    record_id: string;
	    matched: boolean;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new DryRunResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record_id = source["record_id"];
	        this.matched = source["matched"];
	        this.data = source["data"];
	    }
	}

}


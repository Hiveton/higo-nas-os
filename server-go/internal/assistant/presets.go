package assistant

// Preset is a specialized agent role: a system prompt plus a tool-domain scope
// and a few starter tasks. The Agent Workbench launches a thread bound to a
// preset so the real MCP agent runs focused on one job with a limited toolset.
type Preset struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Icon         string   `json:"icon"`
	SystemPrompt string   `json:"systemPrompt"`
	ToolDomains  []string `json:"toolDomains"`
	Starters     []string `json:"starters"`
}

// builtinPresets are the shipped agent roles. ToolDomains map to the MCP tool
// name prefixes (higo_<domain>_...); an empty scope would mean "all tools".
var builtinPresets = []Preset{
	{
		ID:          "storage-admin",
		Name:        "存储管理员",
		Description: "盯存储池、磁盘健康与容量，给出扩容/清理建议。",
		Icon:        "hard-drive",
		SystemPrompt: "你是 HiGoOS 的存储管理员助手。聚焦存储池、磁盘 SMART、容量与监控指标，" +
			"用简洁中文给出诊断与建议。涉及创建快照、SMART 扫描等操作前先说明影响并等待用户确认。",
		ToolDomains: []string{"storage", "monitoring"},
		Starters:    []string{"检查存储池健康与剩余容量", "有没有磁盘 SMART 风险？", "给数据卷创建一个快照"},
	},
	{
		ID:          "file-curator",
		Name:        "文件策展",
		Description: "语义检索、整理重复/陈旧文件，给出归档建议。",
		Icon:        "folder-search",
		SystemPrompt: "你是 HiGoOS 的文件策展助手。擅长语义检索文件、发现重复与可归档内容并提出整理方案。" +
			"移动、重命名、删除等操作必须先预览影响并等待用户确认。",
		ToolDomains: []string{"files", "search", "steward"},
		Starters:    []string{"找出重复或可归档的文件", "搜索去年的合同文件", "下载目录里有哪些可以清理"},
	},
	{
		ID:          "devops",
		Name:        "运维监控",
		Description: "看设备指标、Docker 容器与下载任务，处理异常。",
		Icon:        "activity",
		SystemPrompt: "你是 HiGoOS 的运维监控助手。关注 CPU/内存/磁盘/网络指标、Docker 容器与下载任务状态。" +
			"启停容器、暂停/删除任务等操作前先说明影响并等待用户确认。",
		ToolDomains: []string{"monitoring", "docker", "downloads", "storage"},
		Starters:    []string{"现在设备负载怎么样？", "有哪些 Docker 容器在运行？", "下载任务有异常吗？"},
	},
	{
		ID:          "home-archivist",
		Name:        "家庭资料管家",
		Description: "整理证件、保修、照片等家庭资料，提供问答。",
		Icon:        "image",
		SystemPrompt: "你是 HiGoOS 的家庭资料管家。帮助整理证件、保修单、照片与家庭文档，提供检索与问答。" +
			"任何修改类操作前先说明影响并等待用户确认。",
		ToolDomains: []string{"files", "search", "media"},
		Starters:    []string{"找一下家里的保修资料", "整理照片里的重要证件", "搜索体检报告"},
	},
	{
		ID:          "security",
		Name:        "安全巡检",
		Description: "排查公开分享、风险权限与异常，给出整改建议。",
		Icon:        "shield-check",
		SystemPrompt: "你是 HiGoOS 的安全巡检助手。排查公开分享链接、风险权限、缺失 MFA 与异常文件，给出整改建议。" +
			"撤销分享、收紧权限等操作前先说明影响并等待用户确认。",
		ToolDomains: []string{"security", "steward"},
		Starters:    []string{"运行一次安全巡检", "有哪些公开分享链接有风险？", "哪些账号没开启 MFA？"},
	},
}

// Presets returns the shipped agent presets.
func Presets() []Preset {
	return append([]Preset(nil), builtinPresets...)
}

// PresetByID resolves a preset; ok is false when not found.
func PresetByID(id string) (Preset, bool) {
	for _, p := range builtinPresets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

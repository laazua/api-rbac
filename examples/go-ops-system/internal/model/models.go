// 运维系统 — 业务数据模型
// 纯业务实体, 与 RBAC 权限模型完全解耦
package model

// Server 服务器
type Server struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
	CPU    string `json:"cpu"`    // CPU 型号/核数
	Memory string `json:"memory"` // 内存
	Status string `json:"status"` // running / stopped / error
}

// Deployment 发布记录
type Deployment struct {
	ID       int    `json:"id"`
	Project  string `json:"project"`
	Version  string `json:"version"`
	Env      string `json:"env"`     // production / staging
	Operator string `json:"operator"` // 操作人
	Status   string `json:"status"`   // success / failed / rolling
}

// Alert 告警
type Alert struct {
	ID       int    `json:"id"`
	Level    string `json:"level"`    // critical / warning / info
	Source   string `json:"source"`   // 告警来源 (服务器名/服务名)
	Message  string `json:"message"`  // 告警内容
	Time     string `json:"time"`     // 告警时间
	Acked    bool   `json:"acked"`    // 是否已确认
	AckedBy  string `json:"acked_by"` // 确认人
}

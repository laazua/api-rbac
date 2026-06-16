// 运维系统 — 告警管理 Handler
package handler

import (
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"go-ops-system/internal/middleware"
	"go-ops-system/internal/model"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

// AlertHandler 告警管理 Handler
type AlertHandler struct {
	mu     sync.RWMutex
	alerts map[int]*model.Alert
	nextID atomic.Int64
}

// NewAlertHandler 创建 AlertHandler (含模拟数据)
func NewAlertHandler() *AlertHandler {
	h := &AlertHandler{
		alerts: make(map[int]*model.Alert),
	}
	h.alerts[1] = &model.Alert{ID: 1, Level: "critical", Source: "web-01", Message: "CPU 使用率持续 95% 超过 5 分钟", Time: "2026-06-16 10:23:00", Acked: false}
	h.alerts[2] = &model.Alert{ID: 2, Level: "warning", Source: "db-master", Message: "磁盘使用率达到 80%", Time: "2026-06-16 09:45:00", Acked: false}
	h.alerts[3] = &model.Alert{ID: 3, Level: "info", Source: "cache-01", Message: "内存使用率超过 70%", Time: "2026-06-16 08:30:00", Acked: true, AckedBy: "admin"}
	h.alerts[4] = &model.Alert{ID: 4, Level: "warning", Source: "web-02", Message: "Nginx 502 错误频率增加", Time: "2026-06-16 11:00:00", Acked: false}
	h.alerts[5] = &model.Alert{ID: 5, Level: "critical", Source: "api-service", Message: "API 响应时间超过 5 秒", Time: "2026-06-16 11:05:00", Acked: false}
	h.nextID.Store(5)
	return h
}

// List 获取告警列表
// GET /api/alerts  [权限: alert:read]
func (h *AlertHandler) List(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]*model.Alert, 0, len(h.alerts))
	for _, a := range h.alerts {
		list = append(list, a)
	}
	response.Success(c, list)
}

// Ack 确认告警
// POST /api/alerts/ack  [权限: alert:ack]
func (h *AlertHandler) Ack(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	a, ok := h.alerts[req.ID]
	if !ok {
		response.Error(c, errcode.NotFound)
		return
	}
	a.Acked = true
	a.AckedBy = middleware.GetUsername(c)
	response.Success(c, gin.H{"message": "告警 [" + a.Message + "] 已确认"})
}

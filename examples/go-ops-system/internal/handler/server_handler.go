// 运维系统 — 服务器管理 Handler
package handler

import (
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"go-ops-system/internal/model"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

// ServerHandler 服务器管理 Handler
type ServerHandler struct {
	mu      sync.RWMutex
	servers map[int]*model.Server
	nextID  atomic.Int64
}

// NewServerHandler 创建 ServerHandler (含模拟数据)
func NewServerHandler() *ServerHandler {
	h := &ServerHandler{
		servers: make(map[int]*model.Server),
	}
	// 初始化模拟数据
	h.servers[1] = &model.Server{ID: 1, Name: "web-01", IP: "10.0.1.10", CPU: "4核 Intel Xeon", Memory: "16GB", Status: "running"}
	h.servers[2] = &model.Server{ID: 2, Name: "web-02", IP: "10.0.1.11", CPU: "8核 Intel Xeon", Memory: "32GB", Status: "running"}
	h.servers[3] = &model.Server{ID: 3, Name: "db-master", IP: "10.0.2.10", CPU: "16核 AMD EPYC", Memory: "64GB", Status: "running"}
	h.servers[4] = &model.Server{ID: 4, Name: "db-slave", IP: "10.0.2.11", CPU: "8核 AMD EPYC", Memory: "32GB", Status: "stopped"}
	h.servers[5] = &model.Server{ID: 5, Name: "cache-01", IP: "10.0.3.10", CPU: "4核", Memory: "16GB", Status: "error"}
	h.nextID.Store(5)
	return h
}

// List 获取服务器列表
// GET /api/servers  [权限: server:read]
func (h *ServerHandler) List(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]*model.Server, 0, len(h.servers))
	for _, s := range h.servers {
		list = append(list, s)
	}
	response.Success(c, list)
}

// Create 创建服务器
// POST /api/servers  [权限: server:create]
func (h *ServerHandler) Create(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		IP     string `json:"ip" binding:"required"`
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.Lock()
	id := int(h.nextID.Add(1))
	s := &model.Server{
		ID: id, Name: req.Name, IP: req.IP,
		CPU: req.CPU, Memory: req.Memory, Status: "running",
	}
	h.servers[id] = s
	h.mu.Unlock()

	response.Success(c, s)
}

// Delete 删除服务器
// DELETE /api/servers/:id  [权限: server:delete]
func (h *ServerHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.servers[id]; !ok {
		response.Error(c, errcode.NotFound)
		return
	}
	delete(h.servers, id)
	response.Success(c, gin.H{"message": "服务器已删除"})
}

// Restart 重启服务器
// POST /api/servers/restart  [权限: server:restart]
func (h *ServerHandler) Restart(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	s, ok := h.servers[req.ID]
	if !ok {
		response.Error(c, errcode.NotFound)
		return
	}
	s.Status = "running"
	response.Success(c, gin.H{"message": "服务器 " + s.Name + " 重启成功"})
}

// Stop 停止服务器
// POST /api/servers/stop  [权限: server:stop]
func (h *ServerHandler) Stop(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	s, ok := h.servers[req.ID]
	if !ok {
		response.Error(c, errcode.NotFound)
		return
	}
	s.Status = "stopped"
	response.Success(c, gin.H{"message": "服务器 " + s.Name + " 已停止"})
}

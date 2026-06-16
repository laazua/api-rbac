// 运维系统 — 发布管理 Handler
package handler

import (
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"go-ops-system/internal/middleware"
	"go-ops-system/internal/model"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

// DeploymentHandler 发布管理 Handler
type DeploymentHandler struct {
	mu          sync.RWMutex
	deployments map[int]*model.Deployment
	nextID      atomic.Int64
}

// NewDeploymentHandler 创建 DeploymentHandler (含模拟数据)
func NewDeploymentHandler() *DeploymentHandler {
	h := &DeploymentHandler{
		deployments: make(map[int]*model.Deployment),
	}
	h.deployments[1] = &model.Deployment{ID: 1, Project: "web-app", Version: "v2.3.1", Env: "production", Operator: "admin", Status: "success"}
	h.deployments[2] = &model.Deployment{ID: 2, Project: "api-service", Version: "v1.5.0", Env: "staging", Operator: "opsadmin", Status: "failed"}
	h.deployments[3] = &model.Deployment{ID: 3, Project: "data-pipeline", Version: "v3.0.2", Env: "production", Operator: "admin", Status: "success"}
	h.nextID.Store(3)
	return h
}

// List 获取发布列表
// GET /api/deployments  [权限: deployment:read]
func (h *DeploymentHandler) List(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]*model.Deployment, 0, len(h.deployments))
	for _, d := range h.deployments {
		list = append(list, d)
	}
	response.Success(c, list)
}

// Execute 执行发布
// POST /api/deployments  [权限: deployment:execute]
func (h *DeploymentHandler) Execute(c *gin.Context) {
	var req struct {
		Project string `json:"project" binding:"required"`
		Version string `json:"version" binding:"required"`
		Env     string `json:"env" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.Lock()
	id := int(h.nextID.Add(1))
	d := &model.Deployment{
		ID:       id,
		Project:  req.Project,
		Version:  req.Version,
		Env:      req.Env,
		Operator: middleware.GetUsername(c),
		Status:   "success",
	}
	h.deployments[id] = d
	h.mu.Unlock()

	response.Success(c, d)
}

// Rollback 回滚发布
// POST /api/deployments/rollback  [权限: deployment:rollback]
func (h *DeploymentHandler) Rollback(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	d, ok := h.deployments[req.ID]
	if !ok {
		response.Error(c, errcode.NotFound)
		return
	}

	// 创建回滚记录
	rollbackID := int(h.nextID.Add(1))
	rb := &model.Deployment{
		ID:       rollbackID,
		Project:  d.Project,
		Version:  d.Version + " (回滚)",
		Env:      d.Env,
		Operator: middleware.GetUsername(c),
		Status:   "success",
	}
	h.deployments[rollbackID] = rb
	response.Success(c, gin.H{"message": "发布 " + d.Project + " " + d.Version + " 已回滚", "rollback": rb})
}

// GetByID 获取发布详情
// GET /api/deployments/:id  [权限: deployment:read]
func (h *DeploymentHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	d, ok := h.deployments[id]
	if !ok {
		response.Error(c, errcode.NotFound)
		return
	}
	response.Success(c, d)
}

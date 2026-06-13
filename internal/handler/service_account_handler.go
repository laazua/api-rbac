package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"api-rbac/internal/model"
	"api-rbac/internal/service"
	"api-rbac/pkg/errcode"
	"api-rbac/pkg/response"
)

type ServiceAccountHandler struct {
	svc *service.ServiceAccountService
}

func NewServiceAccountHandler(svc *service.ServiceAccountService) *ServiceAccountHandler {
	return &ServiceAccountHandler{svc: svc}
}

// Create 创建服务账号
func (h *ServiceAccountHandler) Create(c *gin.Context) {
	var req model.CreateServiceAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	sa, apiKey, err := h.svc.Create(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"id":      sa.ID,
		"name":    sa.Name,
		"api_key": apiKey,
		"message": "请妥善保管 API Key，此信息仅显示一次",
	})
}

// GetByID 查询单个服务账号
func (h *ServiceAccountHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	sa, err := h.svc.GetByID(uint(id))
	if err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, sa)
}

// List 列表查询
func (h *ServiceAccountHandler) List(c *gin.Context) {
	var req model.ListServiceAccountRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	accounts, total, err := h.svc.List(&req)
	if err != nil {
		response.ErrorWithMsg(c, errcode.InternalError, err.Error())
		return
	}

	response.SuccessWithPage(c, accounts, total, req.Page, req.PageSize)
}

// Update 更新服务账号
func (h *ServiceAccountHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	var req model.UpdateServiceAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, errcode.InvalidParams, err.Error())
		return
	}

	if err := h.svc.Update(uint(id), &req); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除服务账号
func (h *ServiceAccountHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.InvalidParams)
		return
	}

	if err := h.svc.Delete(uint(id)); err != nil {
		response.ErrorWithMsg(c, errcode.NotFound, err.Error())
		return
	}

	response.Success(c, nil)
}

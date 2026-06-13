package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/laazua/api-rbac/pkg/errcode"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errcode.Success,
		Message: errcode.GetMsg(errcode.Success),
		Data:    data,
	})
}

func SuccessWithPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    errcode.Success,
		Message: errcode.GetMsg(errcode.Success),
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

func Error(c *gin.Context, code int) {
	c.JSON(errcode.ToHTTPStatus(code), Response{
		Code:    code,
		Message: errcode.GetMsg(code),
	})
}

func ErrorWithMsg(c *gin.Context, code int, msg string) {
	c.JSON(errcode.ToHTTPStatus(code), Response{
		Code:    code,
		Message: msg,
	})
}

func ErrorWithData(c *gin.Context, code int, data interface{}) {
	c.JSON(errcode.ToHTTPStatus(code), Response{
		Code:    code,
		Message: errcode.GetMsg(code),
		Data:    data,
	})
}

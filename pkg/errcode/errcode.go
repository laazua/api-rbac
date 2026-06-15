package errcode

// 通用错误码
const (
	Success       = 0
	UnknownError  = 1000
	InvalidParams = 1001
	Unauthorized  = 1002
	Forbidden     = 1003
	NotFound      = 1004
	InternalError = 1005
	AlreadyExists = 1006
	TokenExpired  = 1007
	TokenInvalid  = 1008
	PasswordWrong = 1009
	UserDisabled  = 1010
	DBError       = 1011
)

var codeMsg = map[int]string{
	Success:       "success",
	UnknownError:  "未知错误",
	InvalidParams: "参数错误",
	Unauthorized:  "未授权",
	Forbidden:     "无权限",
	NotFound:      "资源不存在",
	InternalError: "服务器内部错误",
	AlreadyExists: "资源已存在",
	TokenExpired:  "Token已过期",
	TokenInvalid:  "Token无效",
	PasswordWrong: "密码错误",
	UserDisabled:  "用户已被禁用",
	DBError:       "数据库错误",
}

func GetMsg(code int) string {
	if msg, ok := codeMsg[code]; ok {
		return msg
	}
	return "未知错误"
}

// ToHTTPStatus 将业务错误码映射为 HTTP 状态码
func ToHTTPStatus(code int) int {
	switch code {
	case Success:
		return 200
	case InvalidParams:
		return 400
	case Unauthorized, TokenExpired, TokenInvalid:
		return 401
	case Forbidden:
		return 403
	case NotFound:
		return 404
	case AlreadyExists:
		return 409
	case InternalError, DBError:
		return 500
	default:
		return 200
	}
}

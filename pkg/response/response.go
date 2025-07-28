package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// PagedResponse 分页响应格式
type PagedResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
	Meta PageMeta    `json:"meta"`
}

// PageMeta 分页元数据
type PageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// 错误码常量
const (
	CodeSuccess           = 200
	CodeBadRequest        = 400
	CodeUnauthorized      = 401
	CodeForbidden         = 403
	CodeNotFound          = 404
	CodeMethodNotAllowed  = 405
	CodeConflict          = 409
	CodeValidationFailed  = 422
	CodeInternalError     = 500
	CodeServiceUnavailable = 503
	
	// 业务错误码
	CodeUserNotFound      = 40001
	CodeUserExists        = 40002
	CodeInvalidPassword   = 40003
	CodeUserDisabled      = 40004
	CodePermissionDenied  = 40005
	CodeTokenExpired      = 40006
	CodeTokenInvalid      = 40007
)

// 错误消息映射
var errorMessages = map[int]string{
	CodeSuccess:           "Success",
	CodeBadRequest:        "Bad Request",
	CodeUnauthorized:      "Unauthorized",
	CodeForbidden:         "Forbidden",
	CodeNotFound:          "Not Found",
	CodeMethodNotAllowed:  "Method Not Allowed",
	CodeConflict:          "Conflict",
	CodeValidationFailed:  "Validation Failed",
	CodeInternalError:     "Internal Server Error",
	CodeServiceUnavailable: "Service Unavailable",
	
	CodeUserNotFound:      "User not found",
	CodeUserExists:        "User already exists",
	CodeInvalidPassword:   "Invalid password",
	CodeUserDisabled:      "User is disabled",
	CodePermissionDenied:  "Permission denied",
	CodeTokenExpired:      "Token expired",
	CodeTokenInvalid:      "Token invalid",
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  "Success",
		Data: data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  message,
		Data: data,
	})
}

// SuccessPaged 分页成功响应
func SuccessPaged(c *gin.Context, data interface{}, meta PageMeta) {
	c.JSON(http.StatusOK, PagedResponse{
		Code: CodeSuccess,
		Msg:  "Success",
		Data: data,
		Meta: meta,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message ...string) {
	httpStatus := getHTTPStatus(code)
	msg := getErrorMessage(code)
	
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message ...string) {
	Error(c, CodeBadRequest, message...)
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context, message ...string) {
	Error(c, CodeUnauthorized, message...)
}

// Forbidden 403错误
func Forbidden(c *gin.Context, message ...string) {
	Error(c, CodeForbidden, message...)
}

// NotFound 404错误
func NotFound(c *gin.Context, message ...string) {
	Error(c, CodeNotFound, message...)
}

// ValidationFailed 422验证失败
func ValidationFailed(c *gin.Context, message ...string) {
	Error(c, CodeValidationFailed, message...)
}

// InternalError 500内部错误
func InternalError(c *gin.Context, message ...string) {
	Error(c, CodeInternalError, message...)
}

// getHTTPStatus 根据业务错误码获取HTTP状态码
func getHTTPStatus(code int) int {
	switch {
	case code == CodeSuccess:
		return http.StatusOK
	case code >= 400 && code < 500:
		return http.StatusBadRequest
	case code >= 40001 && code < 40100:
		switch code {
		case CodeUnauthorized, CodeTokenExpired, CodeTokenInvalid:
			return http.StatusUnauthorized
		case CodeForbidden, CodePermissionDenied:
			return http.StatusForbidden
		case CodeNotFound, CodeUserNotFound:
			return http.StatusNotFound
		case CodeConflict, CodeUserExists:
			return http.StatusConflict
		case CodeValidationFailed:
			return http.StatusUnprocessableEntity
		default:
			return http.StatusBadRequest
		}
	case code >= 500:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// getErrorMessage 获取错误消息
func getErrorMessage(code int) string {
	if msg, exists := errorMessages[code]; exists {
		return msg
	}
	return "Unknown error"
}

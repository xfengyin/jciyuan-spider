// Package errors 提供爬虫全链路统一错误体系，支持错误分类、可重试判断与 errors.As/Is 兼容。
package errors

import (
	"errors"
	"fmt"
)

// Category 错误分类，用于上层决策（重试、熔断、告警、忽略）。
type Category int

const (
	// CategoryUnknown 未知错误
	CategoryUnknown Category = iota
	// CategoryNetwork 网络层错误（超时、连接失败等）
	CategoryNetwork
	// CategoryHTTP HTTP 协议层错误（非 2xx 状态码）
	CategoryHTTP
	// CategoryBlocked 被目标站点拦截（403/验证码/封禁）
	CategoryBlocked
	// CategoryParse 解析错误
	CategoryParse
	// CategoryStorage 存储错误
	CategoryStorage
	// CategoryConfig 配置错误
	CategoryConfig
	// CategoryValidation 数据校验错误
	CategoryValidation
)

// String 返回分类可读名称
func (c Category) String() string {
	switch c {
	case CategoryNetwork:
		return "network"
	case CategoryHTTP:
		return "http"
	case CategoryBlocked:
		return "blocked"
	case CategoryParse:
		return "parse"
	case CategoryStorage:
		return "storage"
	case CategoryConfig:
		return "config"
	case CategoryValidation:
		return "validation"
	default:
		return "unknown"
	}
}

// SpiderError 是 v3 统一错误类型，携带分类、可重试标记与 TraceID。
type SpiderError struct {
	Category Category
	Msg      string
	Err      error
	Retry    bool
	TraceID  string
}

// Error 实现 error 接口
func (e *SpiderError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Category.String(), e.Msg, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Category.String(), e.Msg)
}

// Unwrap 支持 errors.Unwrap
func (e *SpiderError) Unwrap() error {
	return e.Err
}

// Is 支持 errors.Is 按分类与消息匹配
func (e *SpiderError) Is(target error) bool {
	t, ok := target.(*SpiderError)
	if !ok {
		return false
	}
	return e.Category == t.Category && e.Msg == t.Msg
}

// New 创建一个通用 SpiderError
func New(category Category, msg string) *SpiderError {
	return &SpiderError{Category: category, Msg: msg}
}

// Wrap 包装底层 error 并指定分类
func Wrap(err error, category Category, msg string) *SpiderError {
	return &SpiderError{Category: category, Msg: msg, Err: err}
}

// WrapRetryable 包装为可重试错误
func WrapRetryable(err error, category Category, msg string) *SpiderError {
	return &SpiderError{Category: category, Msg: msg, Err: err, Retry: true}
}

// IsCategory 判断错误是否属于指定分类
func IsCategory(err error, category Category) bool {
	var se *SpiderError
	if errors.As(err, &se) {
		return se.Category == category
	}
	return false
}

// IsRetryable 判断错误是否可以重试
func IsRetryable(err error) bool {
	var se *SpiderError
	if errors.As(err, &se) {
		return se.Retry
	}
	// 兜底：网络类错误默认可重试
	return errors.Is(err, ErrNetwork)
}

// 预定义通用错误变量，便于 errors.Is 快速判断
var (
	ErrUnknown    = New(CategoryUnknown, "未知错误")
	ErrNetwork    = New(CategoryNetwork, "网络错误")
	ErrHTTP       = New(CategoryHTTP, "HTTP 错误")
	ErrBlocked    = New(CategoryBlocked, "访问被拦截")
	ErrParse      = New(CategoryParse, "解析错误")
	ErrStorage    = New(CategoryStorage, "存储错误")
	ErrConfig     = New(CategoryConfig, "配置错误")
	ErrValidation = New(CategoryValidation, "数据校验错误")
	ErrNotFound   = New(CategoryValidation, "记录不存在")
	ErrTimeout    = WrapRetryable(nil, CategoryNetwork, "请求超时")
	ErrRetryable  = WrapRetryable(nil, CategoryNetwork, "可重试错误")
)

// BlockedError 表示被目标站点拦截，保留兼容 v2 的字段
// Deprecated: 新代码优先使用 SpiderError + CategoryBlocked
type BlockedError struct {
	URL        string
	StatusCode int
	Message    string
	TraceID    string
}

// Error 实现 error 接口
func (e *BlockedError) Error() string {
	return fmt.Sprintf("Blocked: %s (HTTP %d) - %s", e.URL, e.StatusCode, e.Message)
}

// Is 支持 errors.Is 与 ErrBlocked 匹配
func (e *BlockedError) Is(target error) bool {
	_, ok := target.(*BlockedError)
	return ok
}

// IsBlocked 判断是否为拦截类错误（兼容 v2 类型断言）
func IsBlocked(err error) bool {
	if err == nil {
		return false
	}
	var be *BlockedError
	if errors.As(err, &be) {
		return true
	}
	return IsCategory(err, CategoryBlocked)
}

// CaptchaError 表示遇到验证码页面，应立即停止并告警
type CaptchaError struct {
	URL     string
	Message string
	TraceID string
}

// Error 实现 error 接口
func (e *CaptchaError) Error() string {
	return fmt.Sprintf("Captcha: %s - %s", e.URL, e.Message)
}

// Package log - 日志模块
package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 日志级别
// ============================================================================

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (l Level) Color() string {
	switch l {
	case DEBUG:
		return "\033[36m" // 青色
	case INFO:
		return "\033[32m" // 绿色
	case WARN:
		return "\033[33m" // 黄色
	case ERROR:
		return "\033[31m" // 红色
	default:
		return "\033[0m"
	}
}

const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// ============================================================================
// 日志记录器
// ============================================================================

// Logger 日志记录器
type Logger struct {
	mu         sync.Mutex
	level      Level
	output     io.Writer
	file       *os.File
	module     string
	timeFormat string
}

// NewLogger 创建日志记录器
func NewLogger(module string) *Logger {
	return &Logger{
		level:      INFO,
		output:     os.Stdout,
		module:     module,
		timeFormat: "2006-01-02 15:04:05",
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		l.level = DEBUG
	case "INFO":
		l.level = INFO
	case "WARN":
		l.level = WARN
	case "ERROR":
		l.level = ERROR
	default:
		l.level = INFO
	}
}

// SetOutput 设置输出
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// SetFile 设置日志文件
func (l *Logger) SetFile(path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// 打开文件
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	l.file = f
	return nil
}

// Close 关闭日志文件
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// ============================================================================
// 日志方法
// ============================================================================

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
	os.Exit(1)
}

// ============================================================================
// 内部日志实现
// ============================================================================

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 构建日志内容
	timestamp := time.Now().Format(l.timeFormat)
	levelStr := level.String()
	
	// 获取调用者信息
	_, file, line, ok := runtime.Caller(2)
	caller := ""
	if ok {
		// 简化路径
		parts := strings.Split(file, "/")
		if len(parts) > 2 {
			file = filepath.Join(parts[len(parts)-2:]...)
		}
		caller = fmt.Sprintf("%s:%d", file, line)
	}
	
	// 格式化消息
	message := fmt.Sprintf(format, args...)
	
	// 构建日志行
	var lineStr string
	if caller != "" {
		lineStr = fmt.Sprintf("[%s] [%s] [%s] %s %s\n",
			timestamp, levelStr, l.module, caller, message)
	} else {
		lineStr = fmt.Sprintf("[%s] [%s] [%s] %s\n",
			timestamp, levelStr, l.module, message)
	}
	
	// 彩色输出到控制台
	if l.output == os.Stdout || l.output == os.Stderr {
		colored := fmt.Sprintf("%s%s%s%s [%s]%s [%s%s%s]%s %s%s\n",
			ColorGray, timestamp, ColorReset,
			level.Color(), levelStr, ColorReset,
			ColorBold, l.module, ColorReset,
			caller,
			ColorReset, message)
		fmt.Fprint(l.output, colored)
	} else {
		fmt.Fprint(l.output, lineStr)
	}
	
	// 写入文件
	if l.file != nil {
		fmt.Fprint(l.file, lineStr)
	}
}

// ============================================================================
// 全局日志器
// ============================================================================

var (
	defaultLogger = NewLogger("main")
)

// SetLevel 设置全局日志级别
func SetLevel(level string) {
	defaultLogger.SetLevel(level)
}

// SetFile 设置全局日志文件
func SetFile(path string) error {
	return defaultLogger.SetFile(path)
}

// SetOutput 设置全局输出
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// Close 关闭日志
func Close() {
	defaultLogger.Close()
}

// Debug 调试日志
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info 信息日志
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn 警告日志
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error 错误日志
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal 致命错误
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// GetLogger 获取模块日志器
func GetLogger(module string) *Logger {
	return NewLogger(module)
}

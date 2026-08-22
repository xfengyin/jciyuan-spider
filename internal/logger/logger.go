// Package logger 提供基于 zap + lumberjack 的结构化日志能力，支持 traceId 全链路透传。
package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"jciyuan-spider/internal/model"
)

// Field 是结构化日志字段别名，底层使用 zap.Field。
type Field = zap.Field

// String 构造字符串字段
func String(key, val string) Field { return zap.String(key, val) }

// Int64 构造整数字段
func Int64(key string, val int64) Field { return zap.Int64(key, val) }

// Int 构造整数字段
func Int(key string, val int) Field { return zap.Int(key, val) }

// Strings 构造字符串切片字段
func Strings(key string, val []string) Field { return zap.Strings(key, val) }

// Err 构造错误字段
func Err(err error) Field { return zap.Error(err) }

// Any 构造任意类型字段
func Any(key string, val interface{}) Field { return zap.Any(key, val) }

// Logger 是 v3 日志接口，支持结构化字段与 traceId 链式携带。
type Logger interface {
	// Debug 结构化调试日志
	Debug(msg string, fields ...Field)
	// Info 结构化信息日志
	Info(msg string, fields ...Field)
	// Warn 结构化警告日志
	Warn(msg string, fields ...Field)
	// Error 结构化错误日志
	Error(msg string, fields ...Field)
	// Fatal 结构化致命日志并退出
	Fatal(msg string, fields ...Field)

	// Debugf 兼容 fmt 风格的调试日志
	Debugf(format string, args ...interface{})
	// Infof 兼容 fmt 风格的信息日志
	Infof(format string, args ...interface{})
	// Warnf 兼容 fmt 风格的警告日志
	Warnf(format string, args ...interface{})
	// Errorf 兼容 fmt 风格的错误日志
	Errorf(format string, args ...interface{})
	// Fatalf 兼容 fmt 风格的致命日志并退出
	Fatalf(format string, args ...interface{})

	// With 追加字段并返回新 Logger
	With(fields ...Field) Logger
	// WithTraceID 追加 traceId 字段并返回新 Logger
	WithTraceID(traceID string) Logger
	// WithTrace 从 ctx 提取 traceId 并返回携带该 traceId 的新 Logger
	WithTrace(ctx context.Context) Logger
	// Sync 刷新缓冲
	Sync() error
}

// zapLogger 基于 zap 的 Logger 实现
type zapLogger struct {
	logger  *zap.Logger
	sugar   *zap.SugaredLogger
	traceID string
}

// New 根据配置创建 Logger
func New(cfg model.LogConfig) (Logger, error) {
	core, err := buildCore(cfg)
	if err != nil {
		return nil, fmt.Errorf("构建日志 core 失败: %w", err)
	}
	z := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return &zapLogger{logger: z, sugar: z.Sugar()}, nil
}

// NewFromZap 从已有的 zap.Logger 创建（便于测试）
func NewFromZap(z *zap.Logger) Logger {
	return &zapLogger{logger: z, sugar: z.Sugar()}
}

// buildCore 根据配置构建 zapcore.Core
func buildCore(cfg model.LogConfig) (zapcore.Core, error) {
	level := parseLevel(cfg.Level)
	enableConsole := cfg.Console || cfg.File == ""

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var cores []zapcore.Core
	if enableConsole {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))
	}

	if cfg.File != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.File), 0755); err != nil {
			return nil, err
		}
		ws := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})
		cores = append(cores, zapcore.NewCore(encoder, ws, level))
	}

	return zapcore.NewTee(cores...), nil
}

// parseLevel 解析日志级别字符串
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, injectTraceID(l.traceID, fields)...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, injectTraceID(l.traceID, fields)...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, injectTraceID(l.traceID, fields)...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, injectTraceID(l.traceID, fields)...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, injectTraceID(l.traceID, fields)...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

func (l *zapLogger) With(fields ...Field) Logger {
	newZap := l.logger.With(fields...)
	return &zapLogger{
		logger:  newZap,
		sugar:   newZap.Sugar(),
		traceID: l.traceID,
	}
}

func (l *zapLogger) WithTraceID(traceID string) Logger {
	newZap := l.logger.With(zap.String("trace_id", traceID))
	return &zapLogger{
		logger:  newZap,
		sugar:   newZap.Sugar(),
		traceID: traceID,
	}
}

func (l *zapLogger) WithTrace(ctx context.Context) Logger {
	if traceID, ok := TraceIDFromContext(ctx); ok && traceID != "" {
		return l.WithTraceID(traceID)
	}
	return l
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// injectTraceID 如果存在 traceId 则注入字段
func injectTraceID(traceID string, fields []Field) []Field {
	if traceID == "" {
		return fields
	}
	return append([]Field{zap.String("trace_id", traceID)}, fields...)
}

// traceIDKey 是 context 中 traceId 的私有键类型，避免与其他 string 键冲突。
type traceIDKey struct{}

// WithTraceID 将 traceId 注入 context，供全链路中间件透传。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceIDFromContext 从 context 读取 traceId。
func TraceIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(traceIDKey{}).(string)
	return v, ok
}

// FromContext 从 context 提取 traceId 并返回携带该 traceId 的默认 Logger。
func FromContext(ctx context.Context) Logger {
	return defaultLogger.WithTrace(ctx)
}

// WithTrace 从 context 提取 traceId 并返回携带该 traceId 的默认 Logger。
// 语义上与 FromContext 相同，便于按场景选择调用方。
func WithTrace(ctx context.Context) Logger {
	return FromContext(ctx)
}

// 全局默认日志器，兼容 v2 全局函数调用
var defaultLogger Logger

func init() {
	var err error
	defaultLogger, err = New(model.LogConfig{Level: "info", Console: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化默认日志器失败: %v\n", err)
		os.Exit(1)
	}
}

// SetLevel 设置全局日志级别（仅对后续 New 生效，兼容旧接口）
func SetLevel(level string) {
	// 兼容旧接口：重建默认 logger
	cfg := model.LogConfig{Level: level, Console: true}
	l, err := New(cfg)
	if err == nil {
		defaultLogger = l
	}
}

// SetFile 设置全局日志文件（兼容旧接口）
func SetFile(path string) error {
	cfg := model.LogConfig{Level: "info", File: path, Console: true}
	l, err := New(cfg)
	if err != nil {
		return err
	}
	defaultLogger = l
	return nil
}

// Close 关闭全局日志
func Close() { _ = defaultLogger.Sync() }

// Debug 全局调试日志
func Debug(format string, args ...interface{}) { defaultLogger.Debugf(format, args...) }

// Info 全局信息日志
func Info(format string, args ...interface{}) { defaultLogger.Infof(format, args...) }

// Warn 全局警告日志
func Warn(format string, args ...interface{}) { defaultLogger.Warnf(format, args...) }

// Error 全局错误日志
func Error(format string, args ...interface{}) { defaultLogger.Errorf(format, args...) }

// Fatal 全局致命日志
func Fatal(format string, args ...interface{}) { defaultLogger.Fatalf(format, args...) }

// GetLogger 获取模块日志器（兼容旧接口）
func GetLogger(module string) Logger {
	return defaultLogger.With(zap.String("module", module))
}

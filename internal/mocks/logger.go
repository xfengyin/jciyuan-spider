package mocks

import (
	"context"
	"sync"

	"jciyuan-spider-v2/internal/logger"
)

// LogEntry 记录一次日志调用。
type LogEntry struct {
	Level  string
	Msg    string
	Fields []logger.Field
}

// MockLogger 实现 logger.Logger 接口，用于测试中断言日志输出。
type MockLogger struct {
	mu          sync.RWMutex
	entries     *[]LogEntry
	prefixFields []logger.Field
}

// NewMockLogger 创建 MockLogger 实例。
func NewMockLogger() *MockLogger {
	entries := make([]LogEntry, 0)
	return &MockLogger{entries: &entries}
}

// cloneEntries 返回当前日志条目副本。
func (m *MockLogger) cloneEntries() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]LogEntry, len(*m.entries))
	copy(out, *m.entries)
	return out
}

// append 追加一条日志。
func (m *MockLogger) append(level, msg string, fields []logger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	combined := make([]logger.Field, 0, len(m.prefixFields)+len(fields))
	combined = append(combined, m.prefixFields...)
	combined = append(combined, fields...)
	*m.entries = append(*m.entries, LogEntry{Level: level, Msg: msg, Fields: combined})
}

func (m *MockLogger) Debug(msg string, fields ...logger.Field) { m.append("debug", msg, fields) }
func (m *MockLogger) Info(msg string, fields ...logger.Field)  { m.append("info", msg, fields) }
func (m *MockLogger) Warn(msg string, fields ...logger.Field)  { m.append("warn", msg, fields) }
func (m *MockLogger) Error(msg string, fields ...logger.Field) { m.append("error", msg, fields) }
func (m *MockLogger) Fatal(msg string, fields ...logger.Field) { m.append("fatal", msg, fields) }

func (m *MockLogger) Debugf(format string, args ...interface{}) { m.append("debug", format, nil) }
func (m *MockLogger) Infof(format string, args ...interface{})  { m.append("info", format, nil) }
func (m *MockLogger) Warnf(format string, args ...interface{})  { m.append("warn", format, nil) }
func (m *MockLogger) Errorf(format string, args ...interface{}) { m.append("error", format, nil) }
func (m *MockLogger) Fatalf(format string, args ...interface{}) { m.append("fatal", format, nil) }

// With 返回携带额外字段的新 Logger。
func (m *MockLogger) With(fields ...logger.Field) logger.Logger {
	return &MockLogger{
		entries:      m.entries,
		prefixFields: append(m.clonePrefix(), fields...),
	}
}

// WithTraceID 返回携带 traceId 的新 Logger。
func (m *MockLogger) WithTraceID(traceID string) logger.Logger {
	return m.With(logger.String("trace_id", traceID))
}

// WithTrace 从 ctx 提取 traceId 并返回新 Logger。
func (m *MockLogger) WithTrace(ctx context.Context) logger.Logger {
	if traceID, ok := logger.TraceIDFromContext(ctx); ok && traceID != "" {
		return m.WithTraceID(traceID)
	}
	return m
}

// Sync 刷新缓冲，Mock 实现无操作。
func (m *MockLogger) Sync() error { return nil }

// clonePrefix 返回当前前缀字段副本。
func (m *MockLogger) clonePrefix() []logger.Field {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]logger.Field, len(m.prefixFields))
	copy(out, m.prefixFields)
	return out
}

// Entries 返回所有记录的日志条目。
func (m *MockLogger) Entries() []LogEntry {
	return m.cloneEntries()
}

// HasMessage 检查是否记录过包含指定子串的日志消息。
func (m *MockLogger) HasMessage(sub string) bool {
	for _, e := range m.cloneEntries() {
		if e.Msg == sub {
			return true
		}
	}
	return false
}

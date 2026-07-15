// Package processor 提供字段后处理器接口与内置实现。
package processor

import "context"

// Processor 字段后处理器接口。
type Processor interface {
	// Name 返回处理器名称。
	Name() string
	// Process 对 value 进行处理，支持字符串或字符串切片。
	Process(ctx context.Context, value interface{}) (interface{}, error)
}

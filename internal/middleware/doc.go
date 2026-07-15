// Package middleware 提供 Fetcher 中间件实现，如限流、重试、熔断、代理轮换、trace、日志、指标等。
//
// Handler 与 Middleware 类型定义位于 internal/fetcher 包，以避免 fetcher 与 middleware 之间的循环依赖。
// 具体中间件实现可依赖 fetcher 包并返回 fetcher.Middleware。
package middleware

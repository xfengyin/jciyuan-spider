// Package parser - HTML 解析抽象层
package parser

import "jciyuan-spider-v2/internal/model"

// 注：单一实现但保留接口以支持 DI 和未来扩展。
// 当前实现：HTMLParser

// Parser HTML 解析器接口
type Parser interface {
	// ParseAnimeDetail 解析动漫详情页
	ParseAnimeDetail(html string) (*model.AnimeInfo, error)
}

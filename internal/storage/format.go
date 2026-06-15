package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jciyuan-spider-v2/internal/model"
)

// FormatJSON 格式化为 JSON 字符串
func FormatJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatEpisodeList 格式化剧集列表为文本
func FormatEpisodeList(anime *model.AnimeInfo) string {
	if len(anime.Episodes) == 0 {
		return "暂无剧集"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s (%d集)\n\n", anime.Title, anime.EpisodeNum))

	for _, ep := range anime.Episodes {
		vipTag := ""
		if ep.IsVIP {
			vipTag = " [VIP]"
		}
		sb.WriteString(fmt.Sprintf("[%02d] %s%s\n", ep.Number, ep.Title, vipTag))
	}
	return sb.String()
}

// FormatM3U8 生成 M3U8 播放列表
func FormatM3U8(anime *model.AnimeInfo) string {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n")
	sb.WriteString(fmt.Sprintf("#PLAYLIST:%s\n", anime.Title))
	sb.WriteString("#GENERATOR:jciyuan-spider\n\n")

	for _, ep := range anime.Episodes {
		if ep.M3U8URL != "" {
			sb.WriteString(fmt.Sprintf("#EXTINF:-1,%s\n", ep.Title))
			sb.WriteString(ep.M3U8URL + "\n\n")
		}
	}
	return sb.String()
}

// SaveM3U8 保存 M3U8 文件
func SaveM3U8(anime *model.AnimeInfo, dir string) error {
	filename := filepath.Join(dir, fmt.Sprintf("%s.m3u8", anime.Title))
	content := FormatM3U8(anime)
	return os.WriteFile(filename, []byte(content), 0644)
}

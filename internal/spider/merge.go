package spider

import (
	"context"
	"sort"

	"jciyuan-spider/internal/model"
)

// mergeWithOld 加载旧数据并与新解析数据合并，实现增量更新。
func (s *Spider) mergeWithOld(ctx context.Context, anime *model.AnimeInfo) (*model.AnimeInfo, error) {
	old, err := s.storage.Load(ctx, anime.ID)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return anime, nil
	}
	return s.mergeAnime(old, anime), nil
}

// mergeAnime 合并旧动漫数据与新动漫数据，保留已有的 M3U8 等附加信息。
func (s *Spider) mergeAnime(old, newest *model.AnimeInfo) *model.AnimeInfo {
	merged := *newest
	merged.Episodes = mergeEpisodes(old.Episodes, newest.Episodes)
	merged.EpisodeNum = len(merged.Episodes)
	return &merged
}

// mergeEpisodes 按集数合并剧集列表，旧剧集中已抓取的 M3U8 会被保留。
func mergeEpisodes(old, updated []model.Episode) []model.Episode {
	existing := make(map[int]model.Episode, len(old))
	for _, ep := range old {
		existing[ep.Number] = ep
	}

	for _, ep := range updated {
		if oldEp, ok := existing[ep.Number]; ok && ep.M3U8URL == "" {
			ep.M3U8URL = oldEp.M3U8URL
			ep.IsCrawled = oldEp.IsCrawled
		}
		existing[ep.Number] = ep
	}

	numbers := make([]int, 0, len(existing))
	for n := range existing {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)

	merged := make([]model.Episode, 0, len(numbers))
	for _, n := range numbers {
		merged = append(merged, existing[n])
	}
	return merged
}

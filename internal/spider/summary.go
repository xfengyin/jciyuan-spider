package spider

import (
	"jciyuan-spider/internal/logger"
	"jciyuan-spider/internal/model"
)

// showSummary 显示爬取摘要。
func (s *Spider) showSummary(log logger.Logger, anime *model.AnimeInfo) {
	log.Info("爬取摘要",
		logger.String("title", anime.Title),
		logger.String("year", anime.Year),
		logger.String("region", anime.Region),
		logger.Strings("tags", anime.Tags),
		logger.Int("episode_num", anime.EpisodeNum),
	)

	if len(anime.Episodes) > 0 {
		for i := 0; i < 5 && i < len(anime.Episodes); i++ {
			ep := anime.Episodes[i]
			vipTag := ""
			if ep.IsVIP {
				vipTag = " [VIP]"
			}
			log.Info("剧集",
				logger.Int("number", ep.Number),
				logger.String("title", ep.Title),
				logger.String("m3u8_url", ep.M3U8URL),
				logger.String("vip", vipTag),
			)
		}
		if anime.EpisodeNum > 5 {
			log.Info("更多剧集", logger.Int("total", anime.EpisodeNum))
		}
	}
}

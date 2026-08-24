// 示例：抓取 HTML 并导出 Markdown 文件。
// 复用 crawler 框架（默认 HTTP 爬虫）抓取页面，再用内置的轻量 HTML→Markdown
// 转换器（仅标准库 regexp/html，不引入第三方库）生成 .md 文件。
//
// 转换覆盖：标题 h1-h6、段落、链接、图片、无序/有序列表、粗体/斜体、行内代码、
// 代码块 pre、引用 blockquote、换行/分隔线、HTML 实体解码；其余标签一律剥离。
//
// 运行（在仓库根目录）：
//
//	go run ./examples/markdown -url https://example.com
//	go run ./examples/markdown -url https://example.com,https://example.org -output ./output/markdown
package main

import (
	"context"
	"flag"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"jciyuan-spider/crawler"
)

var (
	urlFlag   = flag.String("url", "", "抓取目标 URL（多个用逗号分隔）")
	outFlag   = flag.String("output", "", "输出目录（默认 ./output/markdown）")
	concFlag  = flag.Int("concurrency", 3, "并发数")
	retryFlag = flag.Int("max-retry", 2, "失败重试次数")
	quietFlag = flag.Bool("quiet", false, "安静模式")
)

func main() {
	flag.Parse()
	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "请通过 -url 指定抓取目标，例如：\n"+
			"  go run ./examples/markdown -url https://example.com")
		os.Exit(2)
	}

	out := *outFlag
	if out == "" {
		out = "./output/markdown"
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	urls := strings.Split(*urlFlag, ",")
	result, err := crawler.NewEngine(crawler.NewHTTPCrawler(), crawler.Options{
		Concurrency: *concFlag,
		MaxRetry:    *retryFlag,
		Timeout:     10 * time.Second,
		Quiet:       *quietFlag,
	}).Run(context.Background(), urls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "抓取运行失败: %v\n", err)
		os.Exit(1)
	}

	written := 0
	for i, it := range result.Items {
		text, _ := it["text"].(string)
		if text == "" {
			continue
		}
		name := slugName(urls[i])
		path := filepath.Join(out, name+".md")
		if err := os.WriteFile(path, []byte(htmlToMarkdown(text)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "写入 %s 失败: %v\n", path, err)
			continue
		}
		written++
		fmt.Printf("已生成 %s（%d 字节）\n", path, len(text))
	}

	fmt.Printf("完成: 抓取成功=%d 失败=%d 生成 MD=%d 输出目录=%s\n",
		result.Stats.Success, result.Stats.Fail, written, out)
}

// slugName 从 URL 生成安全的文件名（host + path 清洗，保留点号）。
func slugName(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
		return strings.Trim(re.ReplaceAllString(raw, "-"), "-")
	}
	base := u.Host + strings.TrimSuffix(u.Path, "/")
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	slug := strings.Trim(re.ReplaceAllString(base, "-"), "-")
	if slug == "" {
		slug = "index"
	}
	return slug
}

// ---------- HTML → Markdown 转换（仅标准库） ----------

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	rePre    = regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	reHead   = regexp.MustCompile(`(?is)<h([1-6])\b[^>]*>(.*?)</h[1-6]>`)
	reLi     = regexp.MustCompile(`(?is)<li\b[^>]*>(.*?)</li>`)
	reLink   = regexp.MustCompile(`(?is)<a\b[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	reImg    = regexp.MustCompile(`(?is)<img\b[^>]*src=["']([^"']+)["'][^>]*>`)
	reAlt    = regexp.MustCompile(`(?i)alt=["']([^"']*)["']`)
	reBR     = regexp.MustCompile(`(?i)<br\b[^>]*/?>`)
	reHR     = regexp.MustCompile(`(?i)<hr\b[^>]*/?>`)
	reB      = regexp.MustCompile(`(?is)<(?:strong|b)\b[^>]*>(.*?)</(?:strong|b)\b[^>]*>`)
	reI      = regexp.MustCompile(`(?is)<(?:em|i)\b[^>]*>(.*?)</(?:em|i)\b[^>]*>`)
	reCode   = regexp.MustCompile(`(?is)<code\b[^>]*>(.*?)</code\b[^>]*>`)
	reQuote  = regexp.MustCompile(`(?is)<blockquote\b[^>]*>(.*?)</blockquote\b[^>]*>`)
	// 块级标签转为换行（\b 防止误匹配如 <body>/<pre>），其余标签直接剥离
	reBlock  = regexp.MustCompile(`(?i)</?(?:p|div|section|article|header|footer|nav|main|aside|form|table|thead|tbody|tfoot|tr|ul|ol|li|dl|dt|dd|figure|figcaption)\b[^>]*>`)
	reTagAny = regexp.MustCompile(`<[^>]+>`)
	reBlank  = regexp.MustCompile(`\n{3,}`)
	reSpace  = regexp.MustCompile(`[ \t]+\n`)
	rePlace  = regexp.MustCompile(`@@PRE(\d+)@@`)
)

// htmlToMarkdown 将 HTML 文本转换为 Markdown。
func htmlToMarkdown(src string) string {
	s := src

	// 脚本/样式整块剔除（含内容）
	s = reScript.ReplaceAllString(s, "")
	s = reStyle.ReplaceAllString(s, "")

	// 代码块：先占位保护，避免内部被行内转换（如 <code>）二次处理
	var preBlocks []string
	s = rePre.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSpace(rePre.FindStringSubmatch(m)[1])
		preBlocks = append(preBlocks, inner)
		return "\n@@PRE" + strconv.Itoa(len(preBlocks)-1) + "@@\n"
	})

	// 标题
	s = reHead.ReplaceAllStringFunc(s, func(m string) string {
		g := reHead.FindStringSubmatch(m)
		level, _ := strconv.Atoi(g[1])
		return "\n" + strings.Repeat("#", level) + " " + strings.TrimSpace(g[2]) + "\n"
	})

	// 列表项（ul/ol 统一转 "- "，保持演示简单）
	s = reLi.ReplaceAllStringFunc(s, func(m string) string {
		inner := reLi.FindStringSubmatch(m)[1]
		return "\n- " + strings.TrimSpace(inner)
	})

	// 链接与图片
	s = reLink.ReplaceAllStringFunc(s, func(m string) string {
		g := reLink.FindStringSubmatch(m)
		return "[" + strings.TrimSpace(g[2]) + "](" + g[1] + ")"
	})
	s = reImg.ReplaceAllStringFunc(s, func(m string) string {
		src := reImg.FindStringSubmatch(m)[1]
		alt := ""
		if a := reAlt.FindStringSubmatch(m); len(a) > 1 {
			alt = a[1]
		}
		return "![" + alt + "](" + src + ")"
	})

	// 换行 / 分隔线
	s = reBR.ReplaceAllString(s, "\n")
	s = reHR.ReplaceAllString(s, "\n---\n")

	// 行内格式
	s = reB.ReplaceAllString(s, "**$1**")
	s = reI.ReplaceAllString(s, "*$1*")
	s = reCode.ReplaceAllString(s, "`$1`")
	s = reQuote.ReplaceAllStringFunc(s, func(m string) string {
		inner := reQuote.FindStringSubmatch(m)[1]
		lines := strings.Split(strings.TrimSpace(inner), "\n")
		for i, ln := range lines {
			lines[i] = "> " + strings.TrimSpace(ln)
		}
		return "\n" + strings.Join(lines, "\n") + "\n"
	})

	// 块级标签 → 换行；剩余标签剥离
	s = reBlock.ReplaceAllString(s, "\n")
	s = reTagAny.ReplaceAllString(s, "")

	// 还原代码块（内部残留标签一并剥离）
	s = rePlace.ReplaceAllStringFunc(s, func(m string) string {
		idx, _ := strconv.Atoi(rePlace.FindStringSubmatch(m)[1])
		if idx < 0 || idx >= len(preBlocks) {
			return m
		}
		return "```\n" + strings.TrimSpace(reTagAny.ReplaceAllString(preBlocks[idx], "")) + "\n```"
	})

	// 实体解码与空白清理
	s = html.UnescapeString(s)
	s = reSpace.ReplaceAllString(s, "\n")
	s = reBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s) + "\n"
}

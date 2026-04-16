package downloader

import (
	"JJFreeBooks/api"
	"JJFreeBooks/config"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type LogFunc func(format string, args ...any)

type Downloader struct {
	config config.Config
	logf   LogFunc
}

type Summary struct {
	TotalBooks      int
	MatchedBooks    int
	DownloadedBooks int
	SkippedBooks    int
}

const chapterFetchMaxAttempts = 3

func New(cfg config.Config, logf LogFunc) *Downloader {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Downloader{config: cfg, logf: logf}
}

func (d *Downloader) RunDailyTasks() (Summary, error) {
	summary := Summary{}

	d.logf("开始执行每日任务 %s", time.Now().Format("2006-01-02 15:04:05"))
	bookList, err := api.GetBooksList()
	if err != nil {
		return summary, fmt.Errorf("获取小说列表失败: %w", err)
	}

	books := bookList.Data.Data
	summary.TotalBooks = len(books)
	if len(books) == 0 {
		d.logf("今日没有免费小说")
		return summary, nil
	}

	if err := os.MkdirAll("data", 0o755); err != nil {
		return summary, fmt.Errorf("创建数据目录失败: %w", err)
	}

	for _, book := range books {
		d.logf("处理小说《%s》", book.NovelName)
		if !shouldDownloadNovel(book.NovelClass, d.config.NovelFilter) {
			summary.SkippedBooks++
			d.logf("跳过《%s》，未命中过滤器", book.NovelName)
			continue
		}
		summary.MatchedBooks++

		filePath := filepath.Join("data", sanitizeFilename(book.NovelName)+".txt")
		if _, err := os.Stat(filePath); err == nil {
			summary.SkippedBooks++
			d.logf("跳过《%s》，目标文件已存在", book.NovelName)
			d.sleepBook()
			continue
		}

		content, err := d.buildBookContent(book)
		if err != nil {
			return summary, err
		}

		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return summary, fmt.Errorf("写入《%s》失败: %w", book.NovelName, err)
		}

		summary.DownloadedBooks++
		d.logf("已写入 %s", filePath)
		d.sleepBook()
	}

	return summary, nil
}

func (d *Downloader) buildBookContent(book api.NovelData) (string, error) {
	chapterList, err := api.GetChapterList(book.NovelID)
	if err != nil {
		return "", fmt.Errorf("获取《%s》章节列表失败: %w", book.NovelName, err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s\n", book.NovelName))
	builder.WriteString(fmt.Sprintf("作者：%s\n", book.AuthorName))
	builder.WriteString("简介：\n")
	builder.WriteString(fmt.Sprintf("　　📖%s📖\n\n", book.NovelIntroshort))
	builder.WriteString(fmt.Sprintf("　　%s\n", book.NovelClass))
	builder.WriteString(fmt.Sprintf("　　【%s】\n", book.FreeDate))
	builder.WriteString(fmt.Sprintf("\n　　◉ 标签：%s\n", book.Tags))
	builder.WriteString(fmt.Sprintf("　　◉ 字数：%s\n", book.NovelSize))
	if book.NovelStep == "2" {
		builder.WriteString("　　◉ 状态：已完结\n")
	} else {
		builder.WriteString(fmt.Sprintf("　　◉ 状态：%s\n", book.NovelStep))
	}
	builder.WriteString("\n　　————————•————————\n")
	builder.WriteString(fmt.Sprintf("　　%s\n\n", formatNovelIntro(book.NovelIntro)))

	for _, chapter := range chapterList.ChapterList {
		chapterDetail, err := d.fetchChapter(book.NovelID, chapter)
		if err != nil {
			return "", err
		}

		builder.WriteString(fmt.Sprintf("第%s章 %s\n%s\n\n", chapterDetail.ChapterID, chapterDetail.ChapterName, chapterDetail.Content))
		d.sleepChapter()
	}

	content := strings.ReplaceAll(builder.String(), "\n\n　　", "\n　　")
	return content, nil
}

func (d *Downloader) fetchChapter(novelID string, chapter api.Chapter) (api.ChapterDetail, error) {
	d.logf("章节 %s: %s", chapter.ChapterID, chapter.ChapterName)
	var lastErr error
	for attempt := 1; attempt <= chapterFetchMaxAttempts; attempt++ {
		if attempt > 1 {
			d.logf("章节 %s 开始第 %d/%d 次抓取", chapter.ChapterID, attempt, chapterFetchMaxAttempts)
		}

		detail, err := d.fetchChapterOnce(novelID, chapter)
		if err == nil {
			if attempt > 1 {
				d.logf("章节 %s 在第 %d/%d 次抓取时成功", chapter.ChapterID, attempt, chapterFetchMaxAttempts)
			}
			return detail, nil
		}

		lastErr = err
		if attempt == chapterFetchMaxAttempts {
			d.logf("章节 %s 抓取失败，已达到最大重试次数 %d: %v", chapter.ChapterID, chapterFetchMaxAttempts, err)
			break
		}

		delay := d.chapterRetryDelay(attempt)
		d.logf("章节 %s 抓取失败，第 %d/%d 次重试前等待 %s: %v", chapter.ChapterID, attempt, chapterFetchMaxAttempts, delay, err)
		time.Sleep(delay)
	}

	return api.ChapterDetail{}, fmt.Errorf("获取章节 %s 失败，已重试 %d 次: %w", chapter.ChapterName, chapterFetchMaxAttempts, lastErr)
}

func (d *Downloader) fetchChapterOnce(novelID string, chapter api.Chapter) (api.ChapterDetail, error) {
	if chapter.IsVip == 0 {
		return api.GetChapterContent(novelID, chapter.ChapterID)
	}
	if strings.TrimSpace(d.config.Token) == "" {
		d.logf("VIP 章节缺少 token，写入占位内容")
		return api.ChapterDetail{
			ChapterID:   chapter.ChapterID,
			ChapterName: chapter.ChapterName,
			Content:     "<该章节为 VIP 章节，当前未配置 token，已跳过正文抓取>",
		}, nil
	}

	detail, err := api.GetVIPChapterContent(d.config.Token, novelID, chapter.ChapterID)
	if err != nil {
		return api.ChapterDetail{}, fmt.Errorf("获取 VIP 章节 %s 失败: %w", chapter.ChapterName, err)
	}
	return detail, nil
}

func (d *Downloader) chapterRetryDelay(attempt int) time.Duration {
	base := d.config.Intervals.Chapter
	if base < 500 {
		base = 500
	}
	return time.Duration(base*attempt) * time.Millisecond
}

func (d *Downloader) sleepChapter() {
	time.Sleep(time.Duration(d.config.Intervals.Chapter) * time.Millisecond)
}

func (d *Downloader) sleepBook() {
	time.Sleep(time.Duration(d.config.Intervals.Book) * time.Millisecond)
}

func shouldDownloadNovel(novelClass string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if strings.EqualFold(strings.TrimSpace(filter), "all") {
			return true
		}
	}
	for _, filter := range filters {
		filter = strings.TrimSpace(filter)
		if filter != "" && strings.Contains(novelClass, filter) {
			return true
		}
	}
	return false
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

func formatNovelIntro(intro string) string {
	if intro == "" {
		return intro
	}

	intro = strings.ReplaceAll(intro, "。”", "XwX1")
	intro = strings.ReplaceAll(intro, "～”", "XwX2")
	intro = strings.ReplaceAll(intro, "～", "～\n　　")
	intro = strings.ReplaceAll(intro, "。", "。\n　　")
	intro = strings.ReplaceAll(intro, "”", "”\n　　")
	intro = strings.ReplaceAll(intro, "\"", "\"\n　　")
	intro = strings.ReplaceAll(intro, "XwX1", "。”\n　　")
	intro = strings.ReplaceAll(intro, "XwX2", "～”\n　　")

	re := regexp.MustCompile(`(\d+)\.`)
	return re.ReplaceAllString(intro, "\n　　$1.")
}

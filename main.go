package main

import (
	"JJFreeBooks/config"
	"JJFreeBooks/downloader"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	runOnce := flag.Bool("once", false, "执行一次任务后退出")
	flag.Parse()

	printBanner()

	appConfig, err := config.LoadConfig()
	if err != nil {
		fmt.Println("配置加载失败:", err)
		os.Exit(1)
	}

	if *runOnce {
		if err := runOnceWithLogs(appConfig); err != nil {
			fmt.Println("任务执行失败:", err)
			os.Exit(1)
		}
		return
	}

	if err := runScheduler(appConfig); err != nil {
		fmt.Println("调度器启动失败:", err)
		os.Exit(1)
	}
}

func printBanner() {
	fmt.Println("======= 晋江免费小说下载器 =======")
	fmt.Println("项目地址: https://github.com/MEMLTS/JJFreeBooks-Go")
	fmt.Println("版本:", version)
	fmt.Println("构建信息:", commit, "@", date)
	fmt.Println("启动时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("=================================")
}

func runOnceWithLogs(cfg config.Config) error {
	d := downloader.New(cfg, func(format string, args ...any) {
		fmt.Printf(format+"\n", args...)
	})
	summary, err := d.RunDailyTasks()
	if err != nil {
		return err
	}
	fmt.Printf("执行完成，总数 %d，命中过滤器 %d，下载 %d，跳过 %d\n", summary.TotalBooks, summary.MatchedBooks, summary.DownloadedBooks, summary.SkippedBooks)
	return nil
}

func runScheduler(cfg config.Config) error {
	c := cron.New()
	if _, err := c.AddFunc(cfg.Cron, func() {
		if err := runOnceWithLogs(cfg); err != nil {
			fmt.Println("定时任务执行失败:", err)
		}
	}); err != nil {
		return err
	}

	if err := runOnceWithLogs(cfg); err != nil {
		return err
	}

	c.Start()
	defer c.Stop()

	fmt.Println("调度器已启动，按 Ctrl+C 退出")
	select {}
}

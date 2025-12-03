package main

import (
	"fmt"
	"log"
	"time"

	githubreleasedownloader "github.com/sunwu57/github-release-downloader"
)

func main() {
	// 创建客户端
	client, err := githubreleasedownloader.NewClient(
		githubreleasedownloader.WithConcurrency(5),              // 并发下载数量
		githubreleasedownloader.WithBufferSize(8*1024*1024),     // 缓冲区大小（8MB）
		githubreleasedownloader.WithTimeout(30*time.Minute),     // 下载超时
		githubreleasedownloader.WithAutoExtract(true),           // 自动解压
		githubreleasedownloader.WithTargetDir("./"),             // 目标目录
		githubreleasedownloader.WithCheckLatest(true),           // 检查最新版本
		githubreleasedownloader.WithLoggerLevel("info"),         // 日志级别
		githubreleasedownloader.WithProxyURL("127.0.0.1:10388"), // SOCKS5代理（可选）
		githubreleasedownloader.WithShowProgress(true),          // 显示下载进度条
		// githubreleasedownloader.WithAccessToken("your-token"),  // GitHub访问令牌（可选）
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 示例1: 下载最新版本
	// fmt.Println("=== 下载最新版本 ===")
	// path, err := client.DownloadLatestRelease("zyedidia", "eget")
	// if err != nil {
	// 	log.Printf("下载失败: %v", err)
	// } else {
	// 	fmt.Printf("下载成功: %s\n", path)
	// }

	// 示例2: 下载指定版本
	// fmt.Println("\n=== 下载指定版本 ===")
	// path, err = client.DownloadSpecificRelease("golang", "go", "go1.21.0")
	// if err != nil {
	// 	log.Printf("下载失败: %v", err)
	// } else {
	// 	fmt.Printf("下载成功: %s\n", path)
	// }

	// 示例3: 检查版本是否为最新
	fmt.Println("\n=== 检查版本是否为最新 ===")
	isLatest, err := client.IsLatestVersion("zyedidia", "eget", "1.3.4")
	if err != nil {
		log.Printf("检查失败: %v", err)
	} else {
		if isLatest {
			fmt.Println("当前版本是最新版本")
		} else {
			fmt.Println("当前版本不是最新版本")
		}
	}

	// // 示例4: 下载源代码（当没有Release文件时）
	// fmt.Println("\n=== 下载源代码 ===")
	// path, err = client.DownloadSourceCode("golang", "go", "go1.21.0")
	// if err != nil {
	// 	log.Printf("下载失败: %v", err)
	// } else {
	// 	fmt.Printf("下载成功: %s\n", path)
	// }
}

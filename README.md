# GitHub Release Downloader

一个高性能、并发安全的Go库，用于自动下载GitHub Release文件，支持缓冲读写、并发下载、自动解压等功能。

## 特性

- ✅ 自动检测并下载GitHub Release最新版本文件
- ✅ 支持缓冲读写和并发下载以提升性能
- ✅ 使用Context管理并发安全
- ✅ 版本检查，避免重复下载最新版本
- ✅ 自动解压下载的压缩文件（支持zip、tar.gz、gz格式）
- ✅ 自定义文件移动到指定目录
- ✅ 当Release中无打包文件时自动下载源码
- ✅ 支持SOCKS5代理优化网络连接
- ✅ 结构化日志记录

## 安装

```bash
go get github.com/yourusername/github-release-downloader
```

## 使用示例

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github-releasedownloader "github.com/yourusername/github-release-downloader"
)

func main() {
	// 创建客户端
	client, err := githubreleasedownloader.NewClient(
		githubreleasedownloader.WithConcurrency(5),              // 并发下载数量
		githubreleasedownloader.WithBufferSize(8*1024*1024),     // 缓冲区大小（8MB）
		githubreleasedownloader.WithTimeout(30*time.Minute),     // 下载超时
		githubreleasedownloader.WithAutoExtract(true),           // 自动解压
		githubreleasedownloader.WithTargetDir("/tmp/downloads"), // 目标目录
		githubreleasedownloader.WithCheckLatest(true),           // 检查最新版本
		githubreleasedownloader.WithLoggerLevel("info"),         // 日志级别
		// githubreleasedownloader.WithProxyURL("127.0.0.1:1080"), // SOCKS5代理（可选）
		// githubreleasedownloader.WithAccessToken("your-token"),  // GitHub访问令牌（可选）
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 下载最新版本
	path, err := client.DownloadLatestRelease("golang", "go")
	if err != nil {
		log.Printf("下载失败: %v", err)
	} else {
		fmt.Printf("下载成功: %s\n", path)
	}

	// 下载指定版本
	path, err = client.DownloadSpecificRelease("golang", "go", "go1.21.0")
	if err != nil {
		log.Printf("下载失败: %v", err)
	} else {
		fmt.Printf("下载成功: %s\n", path)
	}

	// 检查版本是否为最新
	isLatest, err := client.IsLatestVersion("golang", "go", "go1.21.0")
	if err != nil {
		log.Printf("检查失败: %v", err)
	} else {
		if isLatest {
			fmt.Println("当前版本是最新版本")
		} else {
			fmt.Println("当前版本不是最新版本")
		}
	}

	// 下载源代码
	path, err = client.DownloadSourceCode("golang", "go", "go1.21.0")
	if err != nil {
		log.Printf("下载失败: %v", err)
	} else {
		fmt.Printf("下载成功: %s\n", path)
	}
}
```

## API 文档

### Client

`Client` 是库的主要入口点，提供了下载GitHub Release的所有功能。

#### 创建客户端

```go
client, err := githubreleasedownloader.NewClient(options...)
```

#### 选项

- `WithConcurrency(n int)`: 设置并发下载数量
- `WithBufferSize(size int)`: 设置缓冲区大小
- `WithCacheDir(dir string)`: 设置缓存目录
- `WithTimeout(timeout time.Duration)`: 设置下载超时
- `WithProxyURL(url string)`: 设置SOCKS5代理URL
- `WithAutoExtract(extract bool)`: 设置是否自动解压
- `WithTargetDir(dir string)`: 设置目标目录
- `WithDownloadSource(download bool)`: 设置当没有Release文件时是否下载源码
- `WithCheckLatest(check bool)`: 设置是否检查最新版本
- `WithLoggerLevel(level string)`: 设置日志级别
- `WithAccessToken(token string)`: 设置GitHub访问令牌

#### 方法

- `DownloadLatestRelease(owner, repo string) (string, error)`: 下载最新版本的Release
- `DownloadSpecificRelease(owner, repo, tag string) (string, error)`: 下载指定版本的Release
- `IsLatestVersion(owner, repo, currentVersion string) (bool, error)`: 检查当前版本是否为最新版本
- `DownloadSourceCode(owner, repo, tag string) (string, error)`: 下载源代码
- `Close() error`: 关闭客户端

## 性能优化

1. **缓冲读写**: 使用带缓冲的IO操作提高读写性能
2. **并发下载**: 并行下载多个资源
3. **连接复用**: 复用HTTP连接减少握手开销
4. **版本检查**: 避免重复下载最新版本
5. **SOCKS5代理**: 支持代理以优化网络连接

## 错误处理

库使用结构化的错误处理，所有错误都会包含详细的上下文信息。建议在使用时适当处理错误。

## 日志

库使用zap日志库进行结构化日志记录，支持多种日志级别：debug、info、warn、error。

## 依赖

- `github.com/google/go-github/v76/github`: GitHub API交互
- `golang.org/x/net/proxy`: SOCKS5代理支持
- `go.uber.org/zap`: 结构化日志
- `golang.org/x/oauth2`: OAuth2认证

## 许可证

MIT

## 贡献

欢迎提交Issue和Pull Request！
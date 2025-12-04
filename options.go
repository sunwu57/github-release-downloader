package githubreleasedownloader

import (
	"time"
)

// Option 定义函数类型，用于配置Client
type Option func(*Options)

// Options 包含库的所有配置选项
type Options struct {
	Concurrency    int           // 并发下载数量
	BufferSize     int           // 缓冲区大小（字节）
	CacheDir       string        // 缓存目录
	Timeout        time.Duration // 下载超时
	ProxyURL       string        // SOCKS5代理URL
	AutoExtract    bool          // 是否自动解压
	TargetDir      string        // 目标目录
	DownloadSource bool          // 当没有Release文件时是否下载源码
	CheckLatest    bool          // 是否检查最新版本
	LoggerLevel    string        // 日志级别
	AccessToken    string        // GitHub访问令牌
	ShowProgress   bool          // 是否显示下载进度条
}

// 默认选项值
const (
	DefaultConcurrency = 5
	DefaultBufferSize  = 8 * 1024 * 1024 // 8MB
	DefaultTimeout     = 30 * time.Minute
	DefaultLoggerLevel = "info"
)

// 默认选项
func defaultOptions() *Options {
	return &Options{
		Concurrency:    DefaultConcurrency,
		BufferSize:     DefaultBufferSize,
		Timeout:        DefaultTimeout,
		AutoExtract:    false,
		DownloadSource: true,
		CheckLatest:    true,
		LoggerLevel:    DefaultLoggerLevel,
		ShowProgress:   false,
	}
}

// WithConcurrency 设置并发下载数量
func WithConcurrency(n int) Option {
	return func(o *Options) {
		o.Concurrency = n
	}
}

// WithBufferSize 设置缓冲区大小
func WithBufferSize(size int) Option {
	return func(o *Options) {
		o.BufferSize = size
	}
}

// WithCacheDir 设置缓存目录
func WithCacheDir(dir string) Option {
	return func(o *Options) {
		o.CacheDir = dir
	}
}

// WithTimeout 设置下载超时
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.Timeout = timeout
	}
}

// WithProxyURL 设置SOCKS5代理URL
func WithProxyURL(url string) Option {
	return func(o *Options) {
		o.ProxyURL = url
	}
}

// WithAutoExtract 设置是否自动解压
func WithAutoExtract(extract bool) Option {
	return func(o *Options) {
		o.AutoExtract = extract
	}
}

// WithTargetDir 设置目标目录
func WithTargetDir(dir string) Option {
	return func(o *Options) {
		o.TargetDir = dir
	}
}

// WithDownloadSource 设置当没有Release文件时是否下载源码
func WithDownloadSource(download bool) Option {
	return func(o *Options) {
		o.DownloadSource = download
	}
}

// WithCheckLatest 设置是否检查最新版本
func WithCheckLatest(check bool) Option {
	return func(o *Options) {
		o.CheckLatest = check
	}
}

// WithLoggerLevel 设置日志级别
func WithLoggerLevel(level string) Option {
	return func(o *Options) {
		o.LoggerLevel = level
	}
}

// WithAccessToken 设置GitHub访问令牌
func WithAccessToken(token string) Option {
	return func(o *Options) {
		o.AccessToken = token
	}
}

// WithShowProgress 设置是否显示下载进度条
func WithShowProgress(show bool) Option {
	return func(o *Options) {
		o.ShowProgress = show
	}
}

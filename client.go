package githubreleasedownloader

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v76/github"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/proxy"
	"golang.org/x/oauth2"
)

// Downloader 定义下载接口
type Downloader interface {
	// DownloadLatestRelease 下载最新版本的Release
	DownloadLatestRelease(owner, repo string) (string, error)

	// DownloadSpecificRelease 下载指定版本的Release
	DownloadSpecificRelease(owner, repo, tag string) (string, error)

	// IsLatestVersion 检查当前版本是否为最新版本
	IsLatestVersion(owner, repo, currentVersion string) (bool, error)

	// DownloadSourceCode 下载源代码
	DownloadSourceCode(owner, repo, tag string) (string, error)
}

// Client 是库的主要入口点
type Client struct {
	httpClient   *http.Client
	githubClient *github.Client
	options      *Options
	logger       *zap.Logger
}

// NewClient 创建一个新的客户端实例
func NewClient(opts ...Option) (*Client, error) {
	// 应用默认选项
	options := defaultOptions()

	// 应用用户提供的选项
	for _, opt := range opts {
		opt(options)
	}

	// 设置日志
	logger, err := setupLogger(options.LoggerLevel)
	if err != nil {
		return nil, fmt.Errorf("设置日志失败: %w", err)
	}

	// 创建HTTP客户端
	httpClient, err := createHTTPClient(options)
	if err != nil {
		logger.Error("创建HTTP客户端失败", zap.Error(err))
		return nil, fmt.Errorf("创建HTTP客户端失败: %w", err)
	}

	// 创建GitHub客户端
	githubClient := createGitHubClient(httpClient, options.AccessToken)

	// 设置缓存目录
	if options.CacheDir == "" {
		cacheDir, err := getDefaultCacheDir()
		if err != nil {
			logger.Error("获取默认缓存目录失败", zap.Error(err))
			return nil, fmt.Errorf("获取默认缓存目录失败: %w", err)
		}
		options.CacheDir = cacheDir
	}

	// 确保缓存目录存在
	if err := ensureDirExists(options.CacheDir); err != nil {
		logger.Error("创建缓存目录失败", zap.Error(err))
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}

	// 如果设置了目标目录，确保它存在
	if options.TargetDir != "" && options.TargetDir != options.CacheDir {
		if err := ensureDirExists(options.TargetDir); err != nil {
			logger.Error("创建目标目录失败", zap.Error(err))
			return nil, fmt.Errorf("创建目标目录失败: %w", err)
		}
	}

	client := &Client{
		httpClient:   httpClient,
		githubClient: githubClient,
		options:      options,
		logger:       logger,
	}

	logger.Info("GitHub Release Downloader 客户端已初始化",
		zap.String("缓存目录", options.CacheDir),
		zap.Int("并发数", options.Concurrency),
		zap.Bool("自动解压", options.AutoExtract),
	)

	return client, nil
}

// setupLogger 设置日志
func setupLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// createHTTPClient 创建HTTP客户端
func createHTTPClient(options *Options) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
	}

	// 如果设置了代理
	if options.ProxyURL != "" {
		dialer, err := proxy.SOCKS5("tcp", options.ProxyURL, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("创建SOCKS5代理失败: %w", err)
		}
		transport.DialContext = dialer.(proxy.ContextDialer).DialContext
	}

	return &http.Client{
		Transport: transport,
		Timeout:   options.Timeout,
	}, nil
}

// createGitHubClient 创建GitHub客户端
func createGitHubClient(httpClient *http.Client, accessToken string) *github.Client {
	if accessToken != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		return github.NewClient(tc)
	}
	return github.NewClient(httpClient)
}

// getDefaultCacheDir 获取默认缓存目录
func getDefaultCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".github-release-downloader", "cache"), nil
}

// ensureDirExists 确保目录存在
func ensureDirExists(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.logger != nil {
		c.logger.Sync()
	}
	return nil
}

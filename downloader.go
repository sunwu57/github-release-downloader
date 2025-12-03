package githubreleasedownloader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/go-github/v76/github"
	"github.com/schollz/progressbar/v3"
	"go.uber.org/zap"
)

// downloadResult 表示下载结果
type downloadResult struct {
	filePath string
	err      error
}

// DownloadLatestRelease 下载最新版本的Release
func (c *Client) DownloadLatestRelease(owner, repo string) (string, error) {
	c.logger.Info("开始下载最新Release",
		zap.String("owner", owner),
		zap.String("repo", repo),
	)

	// 获取最新Release
	release, err := c.getLatestRelease(owner, repo)
	if err != nil {
		return "", err
	}

	// 检查是否需要下载
	if c.options.CheckLatest {
		// 获取缓存的版本信息
		cacheVersionPath := filepath.Join(c.options.CacheDir, fmt.Sprintf("%s-%s-version.txt", owner, repo))
		if _, statErr := os.Stat(cacheVersionPath); statErr == nil {
			// 读取缓存的版本
			cachedVersion, readErr := os.ReadFile(cacheVersionPath)
			if readErr == nil && string(cachedVersion) == release.GetTagName() {
				c.logger.Info("当前已是最新版本，无需下载",
					zap.String("owner", owner),
					zap.String("repo", repo),
					zap.String("version", release.GetTagName()),
				)
				return filepath.Join(c.options.CacheDir, fmt.Sprintf("%s-%s", owner, repo)), nil
			}
		}
	}

	// 获取Release资产
	assets := c.getReleaseAssets(release)

	// 如果没有资产且配置了下载源代码
	if c.shouldDownloadSource(assets) {
		c.logger.Info("没有找到Release资产，开始下载源代码",
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("tag", release.GetTagName()),
		)
		return c.DownloadSourceCode(owner, repo, release.GetTagName())
	}

	// 下载资产
	filePaths, err := c.downloadAssets(assets)
	if err != nil {
		return "", err
	}

	// 如果只有一个文件，直接返回
	if len(filePaths) == 1 {
		// 如果配置了自动解压，解压文件
		if c.options.AutoExtract {
			extractedPath, err := c.extractFile(filePaths[0])
			if err != nil {
				c.logger.Warn("解压文件失败",
					zap.String("filePath", filePaths[0]),
					zap.Error(err),
				)
				// 解压失败不影响返回
			} else {
				filePaths[0] = extractedPath
			}
		}

		// 如果配置了目标目录，移动文件
		if c.options.TargetDir != "" && c.options.TargetDir != c.options.CacheDir {
			targetPath := filepath.Join(c.options.TargetDir, filepath.Base(filePaths[0]))
			if err := c.moveFile(filePaths[0], targetPath); err != nil {
				c.logger.Warn("移动文件失败",
					zap.String("source", filePaths[0]),
					zap.String("target", targetPath),
					zap.Error(err),
				)
				// 移动失败不影响返回
			} else {
				filePaths[0] = targetPath
			}
		}

		// 更新缓存的版本信息
		if c.options.CheckLatest {
			cacheVersionPath := filepath.Join(c.options.CacheDir, fmt.Sprintf("%s-%s-version.txt", owner, repo))
			if err := os.WriteFile(cacheVersionPath, []byte(release.GetTagName()), 0644); err != nil {
				c.logger.Warn("更新缓存版本信息失败",
					zap.String("path", cacheVersionPath),
					zap.Error(err),
				)
			}
		}

		return filePaths[0], nil
	}

	// 如果有多个文件，返回目录
	dirPath := filepath.Join(c.options.CacheDir, fmt.Sprintf("%s-%s-%s", owner, repo, release.GetTagName()))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 移动所有文件到目录
	for _, filePath := range filePaths {
		targetPath := filepath.Join(dirPath, filepath.Base(filePath))
		if err := c.moveFile(filePath, targetPath); err != nil {
			c.logger.Warn("移动文件失败",
				zap.String("source", filePath),
				zap.String("target", targetPath),
				zap.Error(err),
			)
			continue
		}

		// 如果配置了自动解压，解压文件
		if c.options.AutoExtract {
			c.extractFile(targetPath)
		}
	}

	// 如果配置了目标目录，移动目录
	if c.options.TargetDir != "" && c.options.TargetDir != c.options.CacheDir {
		targetDirPath := filepath.Join(c.options.TargetDir, filepath.Base(dirPath))
		if err := c.moveFile(dirPath, targetDirPath); err != nil {
			c.logger.Warn("移动目录失败",
				zap.String("source", dirPath),
				zap.String("target", targetDirPath),
				zap.Error(err),
			)
		} else {
			dirPath = targetDirPath
		}
	}

	// 更新缓存的版本信息
	if c.options.CheckLatest {
		cacheVersionPath := filepath.Join(c.options.CacheDir, fmt.Sprintf("%s-%s-version.txt", owner, repo))
		if err := os.WriteFile(cacheVersionPath, []byte(release.GetTagName()), 0644); err != nil {
			c.logger.Warn("更新缓存版本信息失败",
				zap.String("path", cacheVersionPath),
				zap.Error(err),
			)
		}
	}

	return dirPath, nil
}

// DownloadSpecificRelease 下载指定版本的Release
func (c *Client) DownloadSpecificRelease(owner, repo, tag string) (string, error) {
	c.logger.Info("开始下载指定版本Release",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("tag", tag),
	)

	// 获取指定版本的Release
	release, err := c.getReleaseByTag(owner, repo, tag)
	if err != nil {
		return "", err
	}

	// 获取Release资产
	assets := c.getReleaseAssets(release)

	// 如果没有资产且配置了下载源代码
	if c.shouldDownloadSource(assets) {
		c.logger.Info("没有找到Release资产，开始下载源代码",
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("tag", tag),
		)
		return c.DownloadSourceCode(owner, repo, tag)
	}

	// 下载资产
	filePaths, err := c.downloadAssets(assets)
	if err != nil {
		return "", err
	}

	// 如果只有一个文件，直接返回
	if len(filePaths) == 1 {
		// 如果配置了自动解压，解压文件
		if c.options.AutoExtract {
			extractedPath, err := c.extractFile(filePaths[0])
			if err != nil {
				c.logger.Warn("解压文件失败",
					zap.String("filePath", filePaths[0]),
					zap.Error(err),
				)
				// 解压失败不影响返回
			} else {
				filePaths[0] = extractedPath
			}
		}

		// 如果配置了目标目录，移动文件
		if c.options.TargetDir != "" && c.options.TargetDir != c.options.CacheDir {
			targetPath := filepath.Join(c.options.TargetDir, filepath.Base(filePaths[0]))
			if err := c.moveFile(filePaths[0], targetPath); err != nil {
				c.logger.Warn("移动文件失败",
					zap.String("source", filePaths[0]),
					zap.String("target", targetPath),
					zap.Error(err),
				)
				// 移动失败不影响返回
			} else {
				filePaths[0] = targetPath
			}
		}

		return filePaths[0], nil
	}

	// 如果有多个文件，返回目录
	dirPath := filepath.Join(c.options.CacheDir, fmt.Sprintf("%s-%s-%s", owner, repo, tag))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 移动所有文件到目录
	for _, filePath := range filePaths {
		targetPath := filepath.Join(dirPath, filepath.Base(filePath))
		if err := c.moveFile(filePath, targetPath); err != nil {
			c.logger.Warn("移动文件失败",
				zap.String("source", filePath),
				zap.String("target", targetPath),
				zap.Error(err),
			)
			continue
		}

		// 如果配置了自动解压，解压文件
		if c.options.AutoExtract {
			c.extractFile(targetPath)
		}
	}

	// 如果配置了目标目录，移动目录
	if c.options.TargetDir != "" && c.options.TargetDir != c.options.CacheDir {
		targetDirPath := filepath.Join(c.options.TargetDir, filepath.Base(dirPath))
		if err := c.moveFile(dirPath, targetDirPath); err != nil {
			c.logger.Warn("移动目录失败",
				zap.String("source", dirPath),
				zap.String("target", targetDirPath),
				zap.Error(err),
			)
		} else {
			dirPath = targetDirPath
		}
	}

	return dirPath, nil
}

// DownloadSourceCode 下载源代码
func (c *Client) DownloadSourceCode(owner, repo, tag string) (string, error) {
	c.logger.Info("开始下载源代码",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("tag", tag),
	)

	// 获取源代码URL
	url, err := c.getSourceCodeURL(owner, repo, tag)
	if err != nil {
		return "", err
	}

	// 构建文件名
	fileName := fmt.Sprintf("%s-%s-%s.tar.gz", owner, repo, tag)
	filePath := filepath.Join(c.options.CacheDir, fileName)

	// 下载文件
	if err := c.downloadWithBuffer(url, filePath); err != nil {
		return "", fmt.Errorf("下载源代码失败: %w", err)
	}

	// 如果配置了自动解压，解压文件
	if c.options.AutoExtract {
		extractedPath, err := c.extractFile(filePath)
		if err != nil {
			c.logger.Warn("解压源代码失败",
				zap.String("filePath", filePath),
				zap.Error(err),
			)
			// 解压失败不影响返回
		} else {
			filePath = extractedPath
		}
	}

	// 如果配置了目标目录，移动文件
	if c.options.TargetDir != "" && c.options.TargetDir != c.options.CacheDir {
		targetPath := filepath.Join(c.options.TargetDir, filepath.Base(filePath))
		if err := c.moveFile(filePath, targetPath); err != nil {
			c.logger.Warn("移动源代码失败",
				zap.String("source", filePath),
				zap.String("target", targetPath),
				zap.Error(err),
			)
			// 移动失败不影响返回
		} else {
			filePath = targetPath
		}
	}

	return filePath, nil
}

// downloadAssets 并发下载多个资产
func (c *Client) downloadAssets(assets []*github.ReleaseAsset) ([]string, error) {
	c.logger.Info("开始并发下载资产",
		zap.Int("assetCount", len(assets)),
		zap.Int("concurrency", c.options.Concurrency),
	)

	ctx, cancel := context.WithTimeout(context.Background(), c.options.Timeout)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan downloadResult, len(assets))
	semaphore := make(chan struct{}, c.options.Concurrency)

	// 启动goroutine下载每个资产
	for _, asset := range assets {
		wg.Add(1)
		go func(a *github.ReleaseAsset) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				results <- downloadResult{err: ctx.Err()}
				return
			default:
			}

			// 下载资产
			filePath, err := c.downloadAsset(a)
			results <- downloadResult{filePath: filePath, err: err}
		}(asset)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var filePaths []string
	var errors []error

	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
			continue
		}
		filePaths = append(filePaths, result.filePath)
	}

	// 检查是否有错误
	if len(errors) > 0 {
		c.logger.Error("部分资产下载失败",
			zap.Int("total", len(assets)),
			zap.Int("success", len(filePaths)),
			zap.Int("failed", len(errors)),
		)

		// 如果所有下载都失败，返回第一个错误
		if len(filePaths) == 0 {
			return nil, fmt.Errorf("所有资产下载失败: %w", errors[0])
		}
	}

	c.logger.Info("资产下载完成",
		zap.Int("total", len(assets)),
		zap.Int("success", len(filePaths)),
		zap.Int("failed", len(errors)),
	)

	return filePaths, nil
}

// downloadAsset 下载单个资产
func (c *Client) downloadAsset(asset *github.ReleaseAsset) (string, error) {
	c.logger.Info("开始下载资产",
		zap.String("name", asset.GetName()),
		zap.Int64("size", int64(asset.GetSize())),
	)

	// 获取下载URL
	url := c.getAssetDownloadURL(asset)

	// 构建文件名和路径
	fileName := asset.GetName()
	filePath := filepath.Join(c.options.CacheDir, fileName)

	// 下载文件
	if err := c.downloadWithBuffer(url, filePath); err != nil {
		c.logger.Error("下载资产失败",
			zap.String("name", asset.GetName()),
			zap.String("url", url),
			zap.Error(err),
		)
		return "", fmt.Errorf("下载资产 %s 失败: %w", asset.GetName(), err)
	}

	c.logger.Info("资产下载成功",
		zap.String("name", asset.GetName()),
		zap.String("path", filePath),
	)

	return filePath, nil
}

// downloadWithBuffer 使用缓冲下载文件
func (c *Client) downloadWithBuffer(url, filePath string) error {
	c.logger.Debug("开始缓冲下载",
		zap.String("url", url),
		zap.String("path", filePath),
		zap.Int("bufferSize", c.options.BufferSize),
	)

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 创建缓冲写入器
	bufferedWriter := bufio.NewWriterSize(file, c.options.BufferSize)
	defer bufferedWriter.Flush()

	// 发送请求
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 获取文件大小
	fileSize := resp.ContentLength

	// 创建缓冲读取器
	bufferedReader := bufio.NewReaderSize(resp.Body, c.options.BufferSize)

	// 开始时间
	startTime := time.Now()
	var totalBytes int64

	// 创建进度条（如果启用）
	var bar *progressbar.ProgressBar
	if c.options.ShowProgress && fileSize > 0 {
		bar = progressbar.DefaultBytes(
			fileSize,
			fmt.Sprintf("下载 %s", filepath.Base(filePath)),
		)
	}

	// 读取并写入数据
	buffer := make([]byte, c.options.BufferSize)
	for {
		n, err := bufferedReader.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("读取数据失败: %w", err)
		}

		if n == 0 {
			break
		}

		if _, err := bufferedWriter.Write(buffer[:n]); err != nil {
			return fmt.Errorf("写入数据失败: %w", err)
		}

		totalBytes += int64(n)

		// 更新进度条
		if bar != nil {
			if _, err := bar.Write(buffer[:n]); err != nil {
				c.logger.Debug("更新进度条失败", zap.Error(err))
			}
		}

		// 每10MB记录一次进度
		if totalBytes%10*1024*1024 == 0 {
			c.logger.Debug("下载进度",
				zap.String("url", url),
				zap.Int64("bytes", totalBytes),
				zap.Duration("elapsed", time.Since(startTime)),
			)
		}
	}

	// 确保所有数据都被写入
	if err := bufferedWriter.Flush(); err != nil {
		return fmt.Errorf("刷新缓冲区失败: %w", err)
	}

	// 关闭进度条
	if bar != nil {
		bar.Close()
	}

	// 计算下载速度
	duration := time.Since(startTime)
	speed := float64(totalBytes) / duration.Seconds() / 1024 / 1024 // MB/s

	c.logger.Info("文件下载完成",
		zap.String("url", url),
		zap.String("path", filePath),
		zap.Int64("size", totalBytes),
		zap.Duration("duration", duration),
		zap.Float64("speed", speed),
	)

	return nil
}
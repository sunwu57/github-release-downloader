package githubreleasedownloader

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/google/go-github/v76/github"
	"go.uber.org/zap"
)

// getLatestRelease 获取最新的Release
func (c *Client) getLatestRelease(owner, repo string) (*github.RepositoryRelease, error) {
	ctx := context.Background()
	
	release, resp, err := c.githubClient.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		c.logger.Error("获取最新Release失败",
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Error(err),
		)
		
		// 检查是否是因为没有Release
		if resp != nil && resp.StatusCode == 404 {
			return nil, fmt.Errorf("仓库 %s/%s 没有Release", owner, repo)
		}
		
		return nil, fmt.Errorf("获取最新Release失败: %w", err)
	}
	
	c.logger.Info("获取最新Release成功",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("tag", release.GetTagName()),
		zap.String("name", release.GetName()),
	)
	
	return release, nil
}

// getReleaseByTag 通过Tag获取Release
func (c *Client) getReleaseByTag(owner, repo, tag string) (*github.RepositoryRelease, error) {
	ctx := context.Background()
	
	release, resp, err := c.githubClient.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		c.logger.Error("通过Tag获取Release失败",
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("tag", tag),
			zap.Error(err),
		)
		
		// 检查是否是因为Tag不存在
		if resp != nil && resp.StatusCode == 404 {
			return nil, fmt.Errorf("仓库 %s/%s 中没有Tag为 %s 的Release", owner, repo, tag)
		}
		
		return nil, fmt.Errorf("通过Tag获取Release失败: %w", err)
	}
	
	c.logger.Info("通过Tag获取Release成功",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("tag", release.GetTagName()),
		zap.String("name", release.GetName()),
	)
	
	return release, nil
}

// getSourceCodeURL 获取源代码URL
func (c *Client) getSourceCodeURL(owner, repo, tag string) (string, error) {
	// 如果没有指定Tag，获取最新的Tag
	if tag == "" {
		release, err := c.getLatestRelease(owner, repo)
		if err != nil {
			return "", err
		}
		tag = release.GetTagName()
	}
	
	// 构建源代码URL
	// GitHub的源代码下载URL格式为: https://github.com/{owner}/{repo}/archive/refs/tags/{tag}.tar.gz
	url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", owner, repo, tag)
	
	c.logger.Info("获取源代码URL成功",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("tag", tag),
		zap.String("url", url),
	)
	
	return url, nil
}

// getLatestTagName 获取最新的Tag名称
func (c *Client) getLatestTagName(owner, repo string) (string, error) {
	release, err := c.getLatestRelease(owner, repo)
	if err != nil {
		return "", err
	}
	
	return release.GetTagName(), nil
}

// IsLatestVersion 检查当前版本是否为最新版本
func (c *Client) IsLatestVersion(owner, repo, currentVersion string) (bool, error) {
	c.logger.Info("检查版本是否为最新",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("currentVersion", currentVersion),
	)
	
	// 获取最新版本
	latestVersion, err := c.getLatestTagName(owner, repo)
	if err != nil {
		return false, err
	}
	
	// 移除版本号前缀的"v"（如果有）
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	latestVersion = strings.TrimPrefix(latestVersion, "v")
	
	// 比较版本号
	isLatest := currentVersion == latestVersion
	
	c.logger.Info("版本检查结果",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("currentVersion", currentVersion),
		zap.String("latestVersion", latestVersion),
		zap.Bool("isLatest", isLatest),
	)
	
	return isLatest, nil
}

// getReleaseAssets 获取Release的所有资产
func (c *Client) getReleaseAssets(release *github.RepositoryRelease) []*github.ReleaseAsset {
	assets := release.Assets
	
	c.logger.Info("获取Release资产",
		zap.String("tag", release.GetTagName()),
		zap.Int("assetCount", len(assets)),
	)
	
	// 如果只有一个资产，直接返回
	if len(assets) <= 1 {
		return assets
	}
	
	// 获取当前操作系统和架构
	currentOS := runtime.GOOS
	currentArch := runtime.GOARCH
	
	c.logger.Info("当前平台信息",
		zap.String("os", currentOS),
		zap.String("arch", currentArch),
	)
	
	// 尝试找到匹配当前平台的资产
	var matchedAssets []*github.ReleaseAsset
	
	for _, asset := range assets {
		name := asset.GetName()
		lowerName := strings.ToLower(name)
		
			// 检查操作系统匹配
		osMatched := false
		archMatched := false
		
		// 操作系统匹配映射
		osMap := map[string][]string{
			"linux":   {"linux", "gnu", "gnulinux"},
			"darwin":  {"darwin", "mac", "osx"},
			"windows": {"windows", "win"},
			"freebsd": {"freebsd", "bsd"},
			"openbsd": {"openbsd", "bsd"},
			"netbsd":  {"netbsd", "bsd"},
		}
		
		// 架构匹配映射
		archMap := map[string][]string{
			"amd64":   {"amd64", "x86_64", "64bit"},
			"386":     {"386", "i386", "x86", "32bit"},
			"arm":     {"arm", "armv5", "armv6", "armv7"},
			"arm64":   {"arm64", "aarch64"},
			"mips":    {"mips"},
			"mipsle":  {"mipsle", "mips32le"},
			"mips64":  {"mips64"},
			"mips64le": {"mips64le"},
			"ppc64":   {"ppc64", "powerpc64"},
			"ppc64le": {"ppc64le", "powerpc64le"},
			"s390x":   {"s390x", "s390"},
		}
		
		// 检查操作系统
		if aliases, exists := osMap[currentOS]; exists {
			for _, alias := range aliases {
				if strings.Contains(lowerName, alias) {
					osMatched = true
					break
				}
			}
		} else {
			// 如果当前OS不在映射中，直接检查是否包含当前OS名称
			osMatched = strings.Contains(lowerName, currentOS)
		}
		
		// 检查架构
		if aliases, exists := archMap[currentArch]; exists {
			for _, alias := range aliases {
				if strings.Contains(lowerName, alias) {
					archMatched = true
					break
				}
			}
		} else {
			// 如果当前架构不在映射中，直接检查是否包含当前架构名称
			archMatched = strings.Contains(lowerName, currentArch)
		}
		
		// 如果操作系统和架构都匹配，添加到匹配列表
		if osMatched && archMatched {
			matchedAssets = append(matchedAssets, asset)
			c.logger.Info("找到匹配的资产",
				zap.String("name", name),
				zap.Bool("osMatched", osMatched),
				zap.Bool("archMatched", archMatched),
			)
		}
	}
	
	// 如果找到匹配的资产，返回匹配的资产
	if len(matchedAssets) > 0 {
		c.logger.Info("找到匹配当前平台的资产",
			zap.String("os", currentOS),
			zap.String("arch", currentArch),
			zap.Int("matchedCount", len(matchedAssets)),
		)
		return matchedAssets
	}
	
	// 如果没有找到匹配的资产，返回第一个资产
	c.logger.Warn("没有找到匹配当前平台的资产，返回第一个资产",
		zap.String("os", currentOS),
		zap.String("arch", currentArch),
		zap.String("assetName", assets[0].GetName()),
	)
	return []*github.ReleaseAsset{assets[0]}
}

// getAssetDownloadURL 获取资产的下载URL
func (c *Client) getAssetDownloadURL(asset *github.ReleaseAsset) string {
	// 优先使用浏览器下载URL，这样可以避免API速率限制
	url := asset.GetBrowserDownloadURL()
	
	c.logger.Debug("获取资产下载URL",
		zap.String("name", asset.GetName()),
		zap.String("url", url),
	)
	
	return url
}

// shouldDownloadSource 检查是否应该下载源代码
func (c *Client) shouldDownloadSource(assets []*github.ReleaseAsset) bool {
	// 如果没有资产，且配置了下载源代码，则返回true
	return len(assets) == 0 && c.options.DownloadSource
}
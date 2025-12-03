package githubreleasedownloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// extractFile 解压文件
func (c *Client) extractFile(filePath string) (string, error) {
	c.logger.Info("开始解压文件",
		zap.String("filePath", filePath),
	)

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	
	var extractedDir string
	var err error

	switch ext {
	case ".zip":
		extractedDir, err = c.extractZip(filePath)
	case ".tar.gz", ".tgz":
		extractedDir, err = c.extractTarGz(filePath)
	case ".gz":
		extractedDir, err = c.extractGz(filePath)
	default:
		return "", fmt.Errorf("不支持的压缩格式: %s", ext)
	}

	if err != nil {
		c.logger.Error("解压文件失败",
			zap.String("filePath", filePath),
			zap.Error(err),
		)
		return "", err
	}

	c.logger.Info("文件解压成功",
		zap.String("filePath", filePath),
		zap.String("extractedDir", extractedDir),
	)

	return extractedDir, nil
}

// extractZip 解压ZIP文件
func (c *Client) extractZip(filePath string) (string, error) {
	// 打开ZIP文件
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开ZIP文件失败: %w", err)
	}
	defer r.Close()

	// 创建解压目录
	extractedDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return "", fmt.Errorf("创建解压目录失败: %w", err)
	}

	// 解压文件
	for _, f := range r.File {
		// 构建目标路径
		targetPath := filepath.Join(extractedDir, f.Name)

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}

		// 如果是目录，跳过
		if f.FileInfo().IsDir() {
			continue
		}

		// 打开源文件
		src, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("打开源文件失败: %w", err)
		}

		// 创建目标文件
		dst, err := os.Create(targetPath)
		if err != nil {
			src.Close()
			return "", fmt.Errorf("创建目标文件失败: %w", err)
		}

		// 复制文件内容
		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()

		if err != nil {
			return "", fmt.Errorf("复制文件内容失败: %w", err)
		}

		// 设置文件权限
		if err := os.Chmod(targetPath, f.Mode()); err != nil {
			c.logger.Warn("设置文件权限失败",
				zap.String("filePath", targetPath),
				zap.Error(err),
			)
		}
	}

	return extractedDir, nil
}

// extractTarGz 解压tar.gz文件
func (c *Client) extractTarGz(filePath string) (string, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建gzip读取器
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gzipReader.Close()

	// 创建tar读取器
	tarReader := tar.NewReader(gzipReader)

	// 创建解压目录
	extractedDir := strings.TrimSuffix(filePath, filepath.Ext(strings.TrimSuffix(filePath, filepath.Ext(filePath))))
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return "", fmt.Errorf("创建解压目录失败: %w", err)
	}

	// 解压文件
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("读取tar文件失败: %w", err)
		}

		// 构建目标路径
		targetPath := filepath.Join(extractedDir, header.Name)

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", fmt.Errorf("创建目录失败: %w", err)
		}

		// 根据文件类型处理
		switch header.Typeflag {
		case tar.TypeDir:
			// 如果是目录，创建目录
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return "", fmt.Errorf("创建目录失败: %w", err)
			}
		case tar.TypeReg:
			// 如果是普通文件，复制内容
			dst, err := os.Create(targetPath)
			if err != nil {
				return "", fmt.Errorf("创建目标文件失败: %w", err)
			}

			_, err = io.Copy(dst, tarReader)
			dst.Close()

			if err != nil {
				return "", fmt.Errorf("复制文件内容失败: %w", err)
			}

			// 设置文件权限
			if err := os.Chmod(targetPath, header.FileInfo().Mode()); err != nil {
				c.logger.Warn("设置文件权限失败",
					zap.String("filePath", targetPath),
					zap.Error(err),
				)
			}
		case tar.TypeSymlink:
			// 如果是符号链接，创建符号链接
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				c.logger.Warn("创建符号链接失败",
					zap.String("targetPath", targetPath),
					zap.String("linkName", header.Linkname),
					zap.Error(err),
				)
			}
		case tar.TypeLink:
			// 如果是硬链接，创建硬链接
			if err := os.Link(header.Linkname, targetPath); err != nil {
				c.logger.Warn("创建硬链接失败",
					zap.String("targetPath", targetPath),
					zap.String("linkName", header.Linkname),
					zap.Error(err),
				)
			}
		}
	}

	return extractedDir, nil
}

// extractGz 解压gz文件
func (c *Client) extractGz(filePath string) (string, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建gzip读取器
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gzipReader.Close()

	// 创建目标文件
	targetPath := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	dst, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	_, err = io.Copy(dst, gzipReader)
	if err != nil {
		return "", fmt.Errorf("复制文件内容失败: %w", err)
	}

	return targetPath, nil
}

// moveFile 移动文件或目录
func (c *Client) moveFile(sourcePath, targetPath string) error {
	c.logger.Info("开始移动文件",
		zap.String("source", sourcePath),
		zap.String("target", targetPath),
	)

	// 检查源文件是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("源文件不存在: %s", sourcePath)
	}

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 检查目标文件是否存在
	if _, err := os.Stat(targetPath); err == nil {
		// 如果目标文件存在，删除它
		if err := os.RemoveAll(targetPath); err != nil {
			return fmt.Errorf("删除目标文件失败: %w", err)
		}
	}

	// 移动文件
	if err := os.Rename(sourcePath, targetPath); err != nil {
		// 如果重命名失败（可能是因为跨设备），尝试复制并删除
		if err := c.copyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("复制文件失败: %w", err)
		}

		// 删除源文件
		if err := os.RemoveAll(sourcePath); err != nil {
			c.logger.Warn("删除源文件失败",
				zap.String("source", sourcePath),
				zap.Error(err),
			)
		}
	}

	c.logger.Info("文件移动成功",
		zap.String("source", sourcePath),
		zap.String("target", targetPath),
	)

	return nil
}

// copyFile 复制文件或目录
func (c *Client) copyFile(sourcePath, targetPath string) error {
	// 获取源文件信息
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 如果是目录
	if sourceInfo.IsDir() {
		// 创建目标目录
		if mkdirErr := os.MkdirAll(targetPath, sourceInfo.Mode()); mkdirErr != nil {
			return fmt.Errorf("创建目标目录失败: %w", mkdirErr)
		}

		// 读取源目录内容
		entries, readDirErr := os.ReadDir(sourcePath)
		if readDirErr != nil {
			return fmt.Errorf("读取源目录内容失败: %w", readDirErr)
		}

		// 复制目录内容
		for _, entry := range entries {
			sourceEntryPath := filepath.Join(sourcePath, entry.Name())
			targetEntryPath := filepath.Join(targetPath, entry.Name())

			if copyErr := c.copyFile(sourceEntryPath, targetEntryPath); copyErr != nil {
				return fmt.Errorf("复制目录项失败: %w", copyErr)
			}
		}

		return nil
	}

	// 如果是文件
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer target.Close()

	// 复制文件内容
	_, err = io.Copy(target, source)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 设置文件权限
	if err := os.Chmod(targetPath, sourceInfo.Mode()); err != nil {
		c.logger.Warn("设置文件权限失败",
			zap.String("filePath", targetPath),
			zap.Error(err),
		)
	}

	return nil
}
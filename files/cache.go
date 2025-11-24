package files

import (
	"fmt"
	"os"
)

const cachePath = "data/cache/"

func init() {
	// 创建缓存目录
	err := createCacheDir()
	if err != nil {
		fmt.Printf("Failed to create cache directory: %v\n", err)
	}
}

// createCacheDir 创建缓存目录
func createCacheDir() error {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return os.MkdirAll(cachePath, 0755)
	}
	return nil
}

// LoadCache 根据名称获取缓存数据
//
// 参数：name - 文件名（不含后缀）
//
// 返回：文件内容的字节数据和错误信息
//
// 如果缓存文件不存在，将创建一个空文件作为初始化
func LoadCache(name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("cache name cannot be empty")
	}

	// 首先尝试从内存缓存获取
	if data, exists := getFromMemoryCache(name); exists {
		return data, nil
	}

	// 从文件系统读取
	filePath := cachePath + name
	if !fileExists(filePath) {
		// 文件不存在，创建空文件作为初始化
		emptyData := []byte{}
		if err := writeFileContent(filePath, emptyData); err != nil {
			return nil, fmt.Errorf("failed to initialize cache file %s: %w", name, err)
		}

		// 更新内存缓存
		updateMemoryCache(name, emptyData)

		return emptyData, nil
	}

	data, err := readFileContent(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file %s: %w", name, err)
	}

	// 更新内存缓存
	updateMemoryCache(name, data)

	return data, nil
}

// SaveCache 保存数据到缓存
//
// 参数：name - 文件名（不含后缀），data - 要保存的字节数据
//
// 返回：错误信息
func SaveCache(name string, data []byte) error {
	if name == "" {
		return fmt.Errorf("cache name cannot be empty")
	}

	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	filePath := cachePath + name

	// 写入文件
	if err := writeFileContent(filePath, data); err != nil {
		return fmt.Errorf("failed to save cache file %s: %w", name, err)
	}

	// 更新内存缓存
	updateMemoryCache(name, data)

	return nil
}

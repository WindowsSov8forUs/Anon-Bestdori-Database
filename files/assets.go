package files

import (
	"fmt"
	"os"
)

const assetsPath = "data/assets/"

func init() {
	// 创建资源目录
	err := createAssetsDir()
	if err != nil {
		return
	}
}

// createAssetsDir 创建资源目录
func createAssetsDir() error {
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		return os.MkdirAll(assetsPath, 0755)
	}
	return nil
}

// GetAssets 根据短路径获取资源数据
//
// 参数：name - 文件名（不含后缀）
//
// 返回：文件内容的字节数据和错误信息
func GetAssets(name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("assets name cannot be empty")
	}

	// 首先尝试从内存缓存获取
	if data, exists := getFromMemoryCache(name); exists {
		return data, nil
	}

	// 从文件系统读取
	filePath := assetsPath + name
	if !fileExists(filePath) {
		return nil, fmt.Errorf("assets file %s does not exist", name)
	}

	data, err := readFileContent(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read assets file %s: %w", name, err)
	}

	// 更新内存缓存
	updateMemoryCache(name, data)

	return data, nil
}

// SaveAssets 保存数据到资源
//
// 参数：name - 文件名（含后缀），data - 要保存的字节数据
//
// 返回：错误信息
func SaveAssets(name string, data []byte) error {
	if name == "" {
		return fmt.Errorf("assets name cannot be empty")
	}

	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	filePath := assetsPath + name

	// 写入文件
	if err := writeFileContent(filePath, data); err != nil {
		return fmt.Errorf("failed to save assets file %s: %w", name, err)
	}

	// 更新内存缓存
	updateMemoryCache(name, data)

	return nil
}

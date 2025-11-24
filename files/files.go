package files

import (
	"io"
	"os"
	"path/filepath"
	"sync"
)

// memoryCache 内存缓存，用于提高性能
var (
	memoryCache = make(map[string][]byte)
	cacheMutex  sync.RWMutex
)

// fileExists 检查文件是否存在
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// readFileContent 读取文件内容
func readFileContent(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// writeFileContent 写入文件内容
func writeFileContent(filePath string, data []byte) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

// updateMemoryCache 更新内存缓存
func updateMemoryCache(name string, data []byte) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// 复制数据以避免外部修改影响缓存
	cachedData := make([]byte, len(data))
	copy(cachedData, data)
	memoryCache[name] = cachedData
}

// getFromMemoryCache 从内存缓存获取数据
func getFromMemoryCache(name string) ([]byte, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	data, exists := memoryCache[name]
	if !exists {
		return nil, false
	}

	// 复制数据以避免外部修改影响缓存
	result := make([]byte, len(data))
	copy(result, data)
	return result, true
}

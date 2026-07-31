package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var Logger = slog.Default()

// Init 初始化日志记录器
func Init(file_path string) error {
	if err := os.MkdirAll(filepath.Dir(file_path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(file_path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	// 使用文本格式输出日志
	Logger = slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, file), nil))
	return nil
}

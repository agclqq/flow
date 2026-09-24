package logger

import (
	"log/slog"
	"os"
)

// New 创建一个结构化日志记录器，输出到标准错误。
func New() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

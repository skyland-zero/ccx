package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置
type Config struct {
	// 日志目录
	LogDir string
	// 日志文件名
	LogFile string
	// 单个日志文件最大大小 (MB)
	MaxSize int
	// 保留的旧日志文件最大数量
	MaxBackups int
	// 保留的旧日志文件最大天数
	MaxAge int
	// 是否压缩旧日志文件
	Compress bool
	// 是否同时输出到控制台
	Console bool
	// 原始日志模式：stdout 保持精简格式，文件写入原始 JSON
	RawLogOutput bool
}

// rawFileLog 仅写文件的 logger，用于 RawLogOutput 模式下的原始 JSON 输出
var rawFileLog *log.Logger

// RawFileLog 返回仅写文件的 logger。
// 未初始化或 RawLogOutput 未开启时回退到全局 logger。
func RawFileLog() *log.Logger {
	if rawFileLog != nil {
		return rawFileLog
	}
	return log.Default()
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		LogDir:     "logs",
		LogFile:    "app.log",
		MaxSize:    100, // 100MB
		MaxBackups: 10,
		MaxAge:     30, // 30 days
		Compress:   true,
		Console:    true,
	}
}

// Setup 初始化日志系统
func Setup(cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 确保日志目录存在
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	logPath := filepath.Join(cfg.LogDir, cfg.LogFile)

	// 配置 lumberjack 日志轮转
	lumberLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	flags := log.Ldate | log.Ltime | log.Lmicroseconds

	if cfg.RawLogOutput {
		// RawLogOutput 模式：log.Printf 仅写 stdout，raw 内容由 RawFileLog 写文件
		if cfg.Console {
			log.SetOutput(os.Stdout)
		} else {
			log.SetOutput(io.Discard)
		}
		rawFileLog = log.New(lumberLogger, "", flags)
	} else {
		// 默认模式：log.Printf 同时写 stdout 和文件
		var writer io.Writer
		if cfg.Console {
			writer = io.MultiWriter(os.Stdout, lumberLogger)
		} else {
			writer = lumberLogger
		}
		log.SetOutput(writer)
		rawFileLog = nil
	}

	log.SetFlags(flags)

	log.Printf("[Logger-Init] 日志系统已初始化")
	log.Printf("[Logger-Init] 日志文件: %s", logPath)
	log.Printf("[Logger-Init] 轮转配置: 最大 %dMB, 保留 %d 个备份, %d 天", cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge)
	if cfg.RawLogOutput {
		log.Printf("[Logger-Init] RawLogOutput 已启用: stdout 精简格式，文件写入原始 JSON")
	}

	return nil
}

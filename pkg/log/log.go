package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"anon-bestdori-database/config"
)

// Logger 日志等级
type LogLevel int

const (
	OFF   LogLevel = iota // 关闭所有日志记录
	FATAL                 // 致命错误
	ERROR                 // 错误
	WARN                  // 警告
	INFO                  // 信息
	DEBUG                 // 调试
	TRACE                 // 追踪
	ALL                   // 所有日志记录
)

// Logger 日志结构体
type Logger struct {
	*logrus.Logger
	Mutex      sync.Mutex
	Level      LogLevel
	lumberjack *lumberjack.Logger
}

// 内部 Logger 对象
var logger Logger

// CustomFormatter 自定义格式化器
type CustomFormatter struct {
	TimestampFormat string
	ForceColors     bool
}

// Format 实现 logrus.Formatter 接口
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	isJournal := os.Getenv("JOURNAL_STREAM") == "1"
	if isJournal {
		// systemd journal 无时间戳和级别（避免冗余）
		var levelColor ColorFunc
		var levelName string
		switch entry.Level {
		case logrus.PanicLevel:
			levelColor = PanicColor
			levelName = "PANIC"
		case logrus.FatalLevel:
			levelColor = FatalColor
			levelName = "FATAL"
		case logrus.ErrorLevel:
			levelColor = ErrorColor
			levelName = "ERROR"
		case logrus.WarnLevel:
			levelColor = WarnColor
			levelName = "WARN"
		case logrus.InfoLevel:
			levelColor = InfoColor
			levelName = "INFO"
		case logrus.DebugLevel:
			levelColor = DebugColor
			levelName = "DEBUG"
		case logrus.TraceLevel:
			levelColor = TraceColor
			levelName = "TRACE"
		default:
			levelColor = White
			levelName = "UNKNOWN"
		}
		level := levelColor("[%s]", levelName)
		return fmt.Appendf(nil, "%s %s\n", level, entry.Message), nil
	}

	// 时间戳颜色（浅蓝色）
	timestamp := HiCyan("[%s]", entry.Time.Format(f.TimestampFormat))

	// 日志级别颜色和名称
	var levelColor ColorFunc
	var levelName string
	switch entry.Level {
	case logrus.PanicLevel:
		levelColor = PanicColor
		levelName = "PANIC"
	case logrus.FatalLevel:
		levelColor = FatalColor
		levelName = "FATAL"
	case logrus.ErrorLevel:
		levelColor = ErrorColor
		levelName = "ERROR"
	case logrus.WarnLevel:
		levelColor = WarnColor
		levelName = "WARN"
	case logrus.InfoLevel:
		levelColor = InfoColor
		levelName = "INFO"
	case logrus.DebugLevel:
		levelColor = DebugColor
		levelName = "DEBUG"
	case logrus.TraceLevel:
		levelColor = TraceColor
		levelName = "TRACE"
	default:
		levelColor = White
		levelName = "UNKNOWN"
	}

	level := levelColor("[%s]", levelName)

	// 组合日志消息
	return []byte(fmt.Sprintf("%s %s: %s\n", timestamp, level, entry.Message)), nil
}

// Init 初始化日志系统
func Init(conf *config.Config, logName string) error {
	// 创建日志目录
	if err := os.MkdirAll("log", 0o755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	logger.Logger = logrus.New()
	logger.SetFormatter(&CustomFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.Join("log", logName+".log"),
		MaxSize:    256, // MB
		MaxBackups: 10,
		MaxAge:     7, // days
		Compress:   true,
		LocalTime:  true,
	}
	logger.lumberjack = lumberjackLogger

	logger.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))

	// 设置级别
	level, err := parseLogLevel(conf.Log.Level)
	if err != nil {
		return err
	}
	logger.Level = level
	logger.SetLevel(convertLogLevel(level))
	return nil
}

// parseLogLevel 解析字符串级别
func parseLogLevel(s string) (LogLevel, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "FATAL":
		return FATAL, nil
	case "ERROR":
		return ERROR, nil
	case "WARN":
		return WARN, nil
	case "INFO":
		return INFO, nil
	case "DEBUG":
		return DEBUG, nil
	case "TRACE":
		return TRACE, nil
	default:
		return INFO, nil // 默认
	}
}

// convertLogLevel 转换自定义日志级别到 Logrus 级别
func convertLogLevel(level LogLevel) logrus.Level {
	switch level {
	case FATAL:
		return logrus.FatalLevel
	case ERROR:
		return logrus.ErrorLevel
	case WARN:
		return logrus.WarnLevel
	case INFO:
		return logrus.InfoLevel
	case DEBUG:
		return logrus.DebugLevel
	case TRACE:
		return logrus.TraceLevel
	default:
		return logrus.InfoLevel
	}
}

// SetLogLevel 设置 Logger 等级
func SetLogLevel(level LogLevel) {
	logger.Level = level
	logger.SetLevel(convertLogLevel(level))
}

func Reload(conf *config.Config) error {
	level, _ := parseLogLevel(conf.Log.Level)
	SetLogLevel(level)
	return nil
}

// GetLogger 获取 Logger 对象
func GetLogger() *Logger {
	return &logger
}

// Info 信息
func Info(v ...any) {
	logger.Info(v...)
}

func Infof(format string, v ...any) {
	logger.Infof(format, v...)
}

// Error 错误
func Error(v ...any) {
	logger.Error(v...)
}

func Errorf(format string, v ...any) {
	logger.Errorf(format, v...)
}

// Warn 警告
func Warn(v ...any) {
	logger.Warn(v...)
}

func Warnf(format string, v ...any) {
	logger.Warnf(format, v...)
}

// Debug 调试
func Debug(v ...any) {
	logger.Debug(v...)
}

func Debugf(format string, v ...any) {
	logger.Debugf(format, v...)
}

// Trace 追踪
func Trace(v ...any) {
	logger.Trace(v...)
}

// Tracef 追踪
func Tracef(format string, v ...any) {
	logger.Tracef(format, v...)
}

// Fatal 致命错误
func Fatal(v ...any) {
	logger.Fatal(v...)
}

func Fatalf(format string, v ...any) {
	logger.Fatalf(format, v...)
}

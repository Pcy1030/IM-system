package logger

import (
	"os"
	"path/filepath"

	"im-system/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *zap.Logger

// InitLogger 初始化日志系统
func InitLogger(cfg config.LogConfig) *zap.Logger {
	// 创建日志目录
	if err := os.MkdirAll(filepath.Dir(cfg.Filename), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}

	// 配置日志级别
	level := getLogLevel(cfg.Level)

	// 配置日志轮转
	writer := &lumberjack.Logger{
		Filename:   cfg.Filename,   // 日志文件路径
		MaxSize:    cfg.MaxSize,    // 单个文件最大大小(MB)
		MaxBackups: cfg.MaxBackups, // 最大备份文件数
		MaxAge:     cfg.MaxAge,     // 最大保存天数
		Compress:   cfg.Compress,   // 是否压缩
	}

	// 创建编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 创建核心配置
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // JSON格式编码器
		zapcore.AddSync(writer),               // 文件输出
		level,                                 // 日志级别
	)

	// 创建日志记录器
	log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	// 替换zap包中的全局logger
	zap.ReplaceGlobals(log)

	return log
}

// getLogLevel 获取日志级别
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	log.Fatal(msg, fields...)
}

// Debugf 格式化调试日志
func Debugf(template string, args ...interface{}) {
	log.Sugar().Debugf(template, args...)
}

// Infof 格式化信息日志
func Infof(template string, args ...interface{}) {
	log.Sugar().Infof(template, args...)
}

// Warnf 格式化警告日志
func Warnf(template string, args ...interface{}) {
	log.Sugar().Warnf(template, args...)
}

// Errorf 格式化错误日志
func Errorf(template string, args ...interface{}) {
	log.Sugar().Errorf(template, args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(template string, args ...interface{}) {
	log.Sugar().Fatalf(template, args...)
}

// WithField 添加字段
func WithField(key string, value interface{}) *zap.Logger {
	return log.With(zap.Any(key, value))
}

// WithFields 添加多个字段
func WithFields(fields map[string]interface{}) *zap.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return log.With(zapFields...)
}

// Sync 同步日志到磁盘
func Sync() error {
	return log.Sync()
}

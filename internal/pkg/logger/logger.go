package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

var (
	Log  *zap.Logger
	atom zap.AtomicLevel
)

// InitLogger 初始化全局日志组件
func InitLogger(logPath string, isDebug bool) {
	atom = zap.NewAtomicLevel()
	if isDebug {
		atom.SetLevel(zap.DebugLevel)
	} else {
		atom.SetLevel(zap.InfoLevel)
	}

	// 配置文件轮转 lumberjack
	logWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // 最大 10 MB
		MaxBackups: 5,  // 保留最多 5 个备份文件
		MaxAge:     30, // 最长保留 30 天
		Compress:   true,
	}

	// JSON 编码器用于写入文件
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logWriter),
		atom,
	)

	// Console 编码器用于控制台（方便开发时查看）
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig),
		zapcore.AddSync(os.Stdout),
		atom,
	)

	// 组合 core
	core := zapcore.NewTee(fileCore, consoleCore)

	// 添加调用者信息和栈追踪
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	// 将原生标准库的 log 劫持并重定向到 Zap (Info级别)
	zap.RedirectStdLog(Log)
}

// SetDebugMode 动态更新日志级别
func SetDebugMode(isDebug bool) {
	if atom != (zap.AtomicLevel{}) {
		if isDebug {
			atom.SetLevel(zap.DebugLevel)
			Log.Info("Log level dynamically updated to: DEBUG")
		} else {
			atom.SetLevel(zap.InfoLevel)
			Log.Info("Log level dynamically updated to: INFO")
		}
	}
}

// ----------------- 便捷方法 -----------------

func Debug(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Warn(msg, fields...)
	}
}

func Error(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Error(msg, fields...)
	}
}

func Fatal(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Fatal(msg, fields...)
	}
}

// Sync 同步缓存到文件
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// ----------------- GORM 日志适配器 -----------------

// GormZapLogger 用于接管 GORM 数据库的 SQL 日志
type GormZapLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	SkipCallerLookup          bool
	IgnoreRecordNotFoundError bool
}

func NewGormLogger(zapLogger *zap.Logger, slowThreshold time.Duration) *GormZapLogger {
	return &GormZapLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  gormlogger.Warn,
		SlowThreshold:             slowThreshold,
		SkipCallerLookup:          false,
		IgnoreRecordNotFoundError: true,
	}
}

// UpdateLevel 根据系统 DebugMode 动态调整 GORM 日志级别
func (l *GormZapLogger) UpdateLevel(isDebug bool) {
	if isDebug {
		l.LogLevel = gormlogger.Info // 记录所有 SQL
	} else {
		l.LogLevel = gormlogger.Warn // 仅记录慢查询和错误
	}
}

func (l *GormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

func (l *GormZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Info(fmt.Sprintf(msg, data...))
	}
}

func (l *GormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.ZapLogger.Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *GormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.ZapLogger.Error(fmt.Sprintf(msg, data...))
	}
}

func (l *GormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!l.IgnoreRecordNotFoundError || err.Error() != "record not found"):
		sql, rows := fc()
		l.ZapLogger.Error("GORM Error", zap.Error(err), zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql), zap.String("source", utils.FileWithLineNum()))
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		sql, rows := fc()
		l.ZapLogger.Warn("GORM Slow SQL", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql), zap.String("source", utils.FileWithLineNum()))
	case l.LogLevel == gormlogger.Info:
		sql, rows := fc()
		l.ZapLogger.Debug("GORM SQL", zap.Duration("elapsed", elapsed), zap.Int64("rows", rows), zap.String("sql", sql), zap.String("source", utils.FileWithLineNum()))
	}
}

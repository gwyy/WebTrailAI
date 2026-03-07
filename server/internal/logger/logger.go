package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/gwyy/WebTrailAI/server/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger interface {
	// 基本日志方法（带键值对）
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	DPanic(args ...interface{}) // Development panic
	Panic(args ...interface{})
	Fatal(args ...interface{})

	// 格式化日志方法（类似于 fmt.Printf）
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	DPanicf(template string, args ...interface{})
	Panicf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})

	// 带键值对的日志方法（zap 的 sugared 风格，支持 key-value pairs）
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	DPanicw(msg string, keysAndValues ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	// 其他常用方法
	Sync() error // 刷新缓冲区
}

type SugaredLogger struct {
	zap *zap.SugaredLogger
}

func NewLogger(cfg config.Config) Logger {
	now := time.Now()
	infoLogFileName := fmt.Sprintf("%s/info/%04d-%02d-%02d.log", cfg.GetString("log.path"), now.Year(), now.Month(), now.Day())
	errorLogFileName := fmt.Sprintf("%s/error/%04d-%02d-%02d.log", cfg.GetString("log.path"), now.Year(), now.Month(), now.Day())
	var coreArr []zapcore.Core

	// 获取编码器
	//encoderConfig := zap.NewProductionEncoderConfig()
	//encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder        // 指定时间格式
	//encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // ，不需要的话取值zapcore.CapitalLevelEncoder就可以了
	////encoderConfig.EncodeCaller = zapcore.FullCallerEncoder        // 显示完整文件路径

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		TimeKey:       "time",
		NameKey:       "name",
		CallerKey:     "file",
		FunctionKey:   "func",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		//EncodeTime: zapcore.ISO8601TimeEncoder, // ISO8601 UTC 时间格式
		//EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		//	enc.AppendInt64(int64(d) / 1000000)
		//},
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		//EncodeCaller: zapcore.FullCallerEncoder,
		//EncodeName:       nil,
		//ConsoleSeparator: "",
	}
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// 日志级别
	highPriority := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zap.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level < zap.ErrorLevel && level >= zap.DebugLevel
	})

	// 当yml配置中的等级大于Error时，lowPriority级别日志停止记录
	if cfg.GetInt("log.level") >= 2 {
		lowPriority = zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return false
		})
	}

	// info文件writeSyncer
	infoFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   infoLogFileName,               //日志文件存放目录，如果文件夹不存在会自动创建
		MaxSize:    cfg.GetInt("log.max-size"),    //文件大小限制,单位MB
		MaxAge:     cfg.GetInt("log.max-age"),     //日志文件保留天数
		MaxBackups: cfg.GetInt("log.max-backups"), //最大保留日志文件数量
		LocalTime:  false,
		Compress:   cfg.GetBool("log.compress"), //是否压缩处理
	})
	// 第三个及之后的参数为写入文件的日志级别,ErrorLevel模式只记录error级别的日志
	infoFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(infoFileWriteSyncer, zapcore.AddSync(os.Stdout)), lowPriority)

	// error文件writeSyncer
	errorFileWriteSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   errorLogFileName,              //日志文件存放目录
		MaxSize:    cfg.GetInt("log.max-size"),    //文件大小限制,单位MB
		MaxAge:     cfg.GetInt("log.max-age"),     //日志文件保留天数
		MaxBackups: cfg.GetInt("log.max-backups"), //最大保留日志文件数量
		LocalTime:  false,
		Compress:   cfg.GetBool("log.compress"), //是否压缩处理
	})
	// 第三个及之后的参数为写入文件的日志级别,ErrorLevel模式只记录error级别的日志
	errorFileCore := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(errorFileWriteSyncer, zapcore.AddSync(os.Stdout)), highPriority)

	coreArr = append(coreArr, infoFileCore)
	coreArr = append(coreArr, errorFileCore)

	logger := zap.New(zapcore.NewTee(coreArr...), zap.AddCaller())

	sugar := logger.Sugar()
	sugarLoger := &SugaredLogger{zap: sugar}
	return sugarLoger
}

// 实现接口的所有方法，通过转发到 sl.zap
func (sl *SugaredLogger) Debug(args ...interface{}) {
	sl.zap.Debug(args...)
}

func (sl *SugaredLogger) Info(args ...interface{}) {
	sl.zap.Info(args...)
}

func (sl *SugaredLogger) Warn(args ...interface{}) {
	sl.zap.Warn(args...)
}

func (sl *SugaredLogger) Error(args ...interface{}) {
	sl.zap.Error(args...)
}

func (sl *SugaredLogger) DPanic(args ...interface{}) {
	sl.zap.DPanic(args...)
}

func (sl *SugaredLogger) Panic(args ...interface{}) {
	sl.zap.Panic(args...)
}

func (sl *SugaredLogger) Fatal(args ...interface{}) {
	sl.zap.Fatal(args...)
}

func (sl *SugaredLogger) Debugf(template string, args ...interface{}) {
	sl.zap.Debugf(template, args...)
}

func (sl *SugaredLogger) Infof(template string, args ...interface{}) {
	sl.zap.Infof(template, args...)
}

func (sl *SugaredLogger) Warnf(template string, args ...interface{}) {
	sl.zap.Warnf(template, args...)
}

func (sl *SugaredLogger) Errorf(template string, args ...interface{}) {
	sl.zap.Errorf(template, args...)
}

func (sl *SugaredLogger) DPanicf(template string, args ...interface{}) {
	sl.zap.DPanicf(template, args...)
}

func (sl *SugaredLogger) Panicf(template string, args ...interface{}) {
	sl.zap.Panicf(template, args...)
}

func (sl *SugaredLogger) Fatalf(template string, args ...interface{}) {
	sl.zap.Fatalf(template, args...)
}

func (sl *SugaredLogger) Debugw(msg string, keysAndValues ...interface{}) {
	sl.zap.Debugw(msg, keysAndValues...)
}

func (sl *SugaredLogger) Infow(msg string, keysAndValues ...interface{}) {
	sl.zap.Infow(msg, keysAndValues...)
}

func (sl *SugaredLogger) Warnw(msg string, keysAndValues ...interface{}) {
	sl.zap.Warnw(msg, keysAndValues...)
}

func (sl *SugaredLogger) Errorw(msg string, keysAndValues ...interface{}) {
	sl.zap.Errorw(msg, keysAndValues...)
}

func (sl *SugaredLogger) DPanicw(msg string, keysAndValues ...interface{}) {
	sl.zap.DPanicw(msg, keysAndValues...)
}

func (sl *SugaredLogger) Panicw(msg string, keysAndValues ...interface{}) {
	sl.zap.Panicw(msg, keysAndValues...)
}

func (sl *SugaredLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	sl.zap.Fatalw(msg, keysAndValues...)
}

func (sl *SugaredLogger) Sync() error {
	return sl.zap.Sync()
}

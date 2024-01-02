package common

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var SugarLogger *zap.SugaredLogger

func InitLogger(filename string, maxSize, maxBackups, maxAge int, compress bool) {
	writeSyncer := getLogWriter(filename, maxSize, maxBackups, maxAge, compress)
	consoleWriteSyncer := zapcore.AddSync(os.Stdout)
	encoder := getEncoder()
	// 创建分级写入器（Tee），日志将同时写入文件和控制台
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel),
		zapcore.NewCore(encoder, consoleWriteSyncer, zapcore.DebugLevel),
	)
	logger := zap.New(core, zap.AddCaller())
	SugarLogger = logger.Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(filename string, maxSize, maxBackups, maxAge int, compress bool) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename, //文件路径
		MaxSize:    maxSize, //日志文件的最大存储量（单位MB），否则切割
		MaxBackups: maxBackups, //最多保留的文件数量
		MaxAge:     maxAge, //旧文件最多保存的天数
		Compress:   compress, //是否压缩
	}
	return zapcore.AddSync(lumberJackLogger)
}


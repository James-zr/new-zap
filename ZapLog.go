package common

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"reflect"
	"runtime"
	"time"
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


//// HttpLogger 请求日志切面
//func HttpLogger() gin.HandlerFunc {
//	return func(c *gin.Context) {
//		// 请求的前置处理
//		start := time.Now()
//		path := c.Request.URL.Path
//
//		// 处理请求
//		c.Next()
//
//		// 请求的后置处理
//		cost := time.Since(start)
//		method := c.Request.Method
//		reqBody := c.Request.Form
//		clientIP := c.ClientIP()
//
//		// 构建日志信息字符串
//		logInfo := fmt.Sprintf(
//			"\n请求地址 : %s"+
//				"\nHTTP METHOD : %s"+
//				"\n请求参数 : %s"+
//				"\nIP : %s"+
//				"\n耗时 : %v",
//			path, method, reqBody, clientIP, cost,
//		)
//
//		// 使用SugarLogger记录日志信息
//		SugarLogger.Info(
//			"\n-------------------请求开始---------------------" +
//			logInfo +
//				"\n-------------------请求结束---------------------",
//			)
//	}
//}


// HttpLogger 请求日志切面
func HttpLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求地址和方法
		path := c.Request.URL.Path
		method := c.Request.Method
		var reqParams interface{}

		// 处理请求前记录时间
		start := time.Now()

		// 创建响应记录器
		recorder := NewResponseRecorder(c.Writer)
		c.Writer = recorder

		// 处理请求
		c.Next()

		// 请求的后置处理
		switch c.ContentType() { // 根据 Content-Type 选择合适的方式来获取请求参数
		case "application/json":
			var jsonMap map[string]interface{}
			if err := c.ShouldBindJSON(&jsonMap); err == nil {
				reqParams = jsonMap
			}
		case "application/x-www-form-urlencoded", "multipart/form-data":
			reqParams = c.Request.Form
		default:
			reqParams = nil //其他类型数据暂不做处理
		}

		cost := time.Since(start)
		clientIP := c.ClientIP()
		// 获取 handler 名称
		handlerName := runtime.FuncForPC(reflect.ValueOf(c.Handler()).Pointer()).Name()
		// 获取响应数据
		responseData := recorder.Body.String()

		// 记录所有需要的信息
		logInfo := fmt.Sprintf(
			"\nCLASS METHOD: %s"+
				"\n请求地址: %s"+
				"\n请求参数 : %s"+
				"\nHTTP METHOD: %s"+
				"\nIP: %s"+
				"\n响应数据: %s"+
				"\n耗时: %v",
			handlerName, path, reqParams, method, clientIP, responseData, cost,
		)

		SugarLogger.Info(
			"\n---------------------------请求开始-----------------------------" +
				logInfo +
				"\n---------------------------请求结束-----------------------------",
				)
	}
}


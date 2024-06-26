package zrLogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"time"
)

var SugarLogger *zap.SugaredLogger

func InitLogger(filename string, maxSize, maxBackups, maxAge int, compress bool) {
	// 确保目录存在且具有正确的权限
	if err := ensureDir(filename); err != nil {
		SugarLogger.Errorf("日志文件创建失败: %v", err)
	}
	
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

func ensureDir(filePath string) error {
	dir := path.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(filename string, maxSize, maxBackups, maxAge int, compress bool) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,   //文件路径
		MaxSize:    maxSize,    //日志文件的最大存储量（单位MB），否则切割
		MaxBackups: maxBackups, //最多保留的文件数量
		MaxAge:     maxAge,     //旧文件最多保存的天数
		Compress:   compress,   //是否压缩
	}

	return zapcore.AddSync(lumberJackLogger)
}

// HttpLogger 请求日志切面
func HttpLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求地址和方法
		path := c.Request.URL.Path
		method := c.Request.Method
		var reqParams interface{}
		var responseData string

		// 处理请求前记录时间
		start := time.Now()

		switch c.GetHeader("Content-Type") { // 根据 Content-Type 选择合适的方式来获取请求参数
		case "application/json":
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				// 处理读取失败的情况
				SugarLogger.Errorf("无法读取请求体：%v", err)
				c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(nil))
			} else {
				// 如果读取成功，则只解析 JSON 请求体
				var jsonMap map[string]interface{}
				if err = json.Unmarshal(bodyBytes, &jsonMap); err == nil {
					reqParams = jsonMap
				} else {
					SugarLogger.Errorf("无法解析 JSON 请求体：%v", err)
				}
				// 重新设置 Body，以供后续中间件或处理函数使用
				c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		case "application/x-www-form-urlencoded", "multipart/form-data":
			reqParams = c.Request.Form
		default:
			reqParams = nil //其他类型数据暂不做处理
		}

		// 创建响应记录器
		recorder := NewResponseRecorder(c.Writer)
		c.Writer = recorder

		// 处理请求
		c.Next()

		respContentType := recorder.ResponseWriter.Header().Get("Content-Type")
		// 解析 Content-Type，移除可选参数
		contentTypeWithoutParams := strings.SplitN(respContentType, ";", 2)[0]

		// 请求的后置处理
		switch contentTypeWithoutParams {
		case "application/json":
			// 处理所有JSON类型的响应，忽略字符集
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, recorder.Body.Bytes(), "", "    "); err == nil {
				responseData = prettyJSON.String()
			} else {
				// 如果 JSON 格式化出错则保持原样
				responseData = recorder.Body.String()
			}
		case "application/x-www-form-urlencoded", "multipart/form-data":
			// 处理form类型的响应
			responseData = recorder.Body.String()
		case "audio/mpeg", "text/html", "application/octet-stream":
			// 忽略记录静态资源或二进制数据的响应
			responseData = "[BINARY DATA]"
		default:
			// 对于未受支持的 Content-Type，不记录详细的请求参数或响应数据
			responseData = "[UNSUPPORTED CONTENT TYPE]"
		}

		cost := time.Since(start)
		clientIP := c.ClientIP()
		// 获取 handler 名称
		handlerName := runtime.FuncForPC(reflect.ValueOf(c.Handler()).Pointer()).Name()

		// 记录所有需要的信息
		logInfo := fmt.Sprintf(
			"\nCLASS METHOD: %s"+
				"\n请求地址: %s"+
				"\n请求参数: %s"+
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

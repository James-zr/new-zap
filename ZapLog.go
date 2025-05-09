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
		// 开始计时
		start := time.Now()

		// 捕获请求地址、方法、IP
		httpPath, method, clientIP := c.Request.URL.Path, c.Request.Method, c.ClientIP()

		// 解析请求参数
		var reqParams interface{}
		switch c.GetHeader("Content-Type") {
		case "application/json":
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			if len(bodyBytes) > 0 {
				var m map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &m); err == nil {
					reqParams = m
				}
			}
			// 恢复 Body
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		case "application/x-www-form-urlencoded", "multipart/form-data":
			_ = c.Request.ParseForm()
			reqParams = c.Request.Form
		default:
			reqParams = "" // 其他类型不记录
		}

		// 设置响应记录器
		recorder := NewResponseRecorder(c.Writer)
		c.Writer = recorder

		// 调用后续处理
		c.Next()

		// 获取状态码
		statusCode := recorder.Status()

		// 解析响应数据
		respCT := strings.SplitN(recorder.Header().Get("Content-Type"), ";", 2)[0]
		var responseData string
		switch respCT {
		case "application/json":
			var buf bytes.Buffer
			if err := json.Indent(&buf, recorder.Body.Bytes(), "", "    "); err == nil {
				responseData = buf.String()
			} else {
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
		handlerName := runtime.FuncForPC(reflect.ValueOf(c.Handler()).Pointer()).Name()
		logInfo := fmt.Sprintf(
			"\nCLASS METHOD: %s"+
				"\n请求地址: %s"+
				"\nHTTP METHOD: %s"+
				"\n请求参数: %v"+ // %v 安全处理 nil、map、结构体
				"\nIP: %s"+
				"\n状态码: %d"+ // 新增状态码
				"\n响应数据: %s"+
				"\n耗时: %v",
			handlerName, httpPath, method, reqParams, clientIP, statusCode, responseData, cost,
		)

		SugarLogger.Info(
			"\n---------------------------请求开始-----------------------------" +
				logInfo +
				"\n---------------------------请求结束-----------------------------",
		)
	}
}

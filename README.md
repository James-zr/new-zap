# new-zap
对zap日志库进行了封装，优化了日志的输出格式，可以自定义日志文件的存储位置、文件数量、大小、保存时间等。

## 使用

以下是一个简单的使用例子：

```go
func main() {
  // 初始化
  zrLogger.InitLogger("LogFile/test.log", 1024, 5, 30, false)
  defer SugarLogger.Sync()
  SugarLogger.Info("测试1")
  SugarLogger.Error("测试2")

  //使用请求日志切面
  route := gin.Default()
  r.Use(zrLogger.HttpLogger())
}
```
## 安装

要安装此库，请运行以下命令：

```shell
go get github.com/James-zr/new-zap
```

- ## 贡献

欢迎任何形式的贡献。如果您发现错误，或者有任何改进建议，请提交 issue 或 pull request。

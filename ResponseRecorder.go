package zrLogger

import (
	"bytes"
	"github.com/gin-gonic/gin"
)

// ResponseRecorder 是一个包装了gin.ResponseWriter的结构体，用于记录响应状态码和响应体
type ResponseRecorder struct {
	gin.ResponseWriter
	Body *bytes.Buffer
}

func NewResponseRecorder(w gin.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{w, &bytes.Buffer{}}
}

//写入方法`Write`同时将数据写入自己的`Body`缓存和原始的`ResponseWriter`中
func (r *ResponseRecorder) Write(b []byte) (int, error) {
	r.Body.Write(b)
	return r.ResponseWriter.Write(b)
}

package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/maxliu9403/common/logger"
)

type httpReqResLog struct {
	Operator   string `json:"operator"`
	URI        string `json:"uri"`
	Method     string `json:"method"`
	Params     string `json:"params"`
	Client     string `json:"client"`
	StatusCode int    `json:"status_code"`
	Response   string `json:"response,omitempty"`
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func GinInterceptor(logResponse bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			bodyBytes []byte
			params    []byte
		)

		// 读取 body（JSON body / Form body）
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		if len(bodyBytes) > 0 {
			params = bodyBytes
		} else {
			_ = c.Request.ParseForm()
			paramsMap := make(map[string]interface{})
			for k, v := range c.Request.Form {
				paramsMap[k] = v
			}
			params, _ = json.Marshal(paramsMap)
		}

		lg := &httpReqResLog{
			Operator: getRequestUser(c.Request.Header),
			URI:      c.Request.RequestURI,
			Method:   c.Request.Method,
			Params:   string(params),
			Client:   c.ClientIP(),
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()

		lg.StatusCode = c.Writer.Status()
		if logResponse {
			lg.Response = blw.body.String()
		}

		logBytes, _ := json.Marshal(&lg)
		logger.Debugf("request details: %s", string(logBytes))
	}
}

func GinInterceptorWithTrace(tra opentracing.Tracer, logResponse bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			bodyBytes []byte
			err       error
			params    []byte
		)

		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		if len(bodyBytes) > 0 {
			params = bodyBytes
		} else {
			_ = c.Request.ParseForm()
			paramsMap := make(map[string]interface{})
			for k, v := range c.Request.Form {
				paramsMap[k] = v
			}
			params, _ = json.Marshal(paramsMap)
		}

		// Tracing span
		var span opentracing.Span
		if tra != nil {
			var spanCtx opentracing.SpanContext
			spanCtx, err = tra.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
			opName := fmt.Sprintf("%s_%s", c.Request.Method, c.Request.URL.Path)
			if err != nil {
				span = tra.StartSpan(opName)
			} else {
				span = tra.StartSpan(opName, opentracing.ChildOf(spanCtx))
			}
			defer span.Finish()
			ext.Component.Set(span, "Gin")
			ext.SpanKindRPCServer.Set(span)

			c.Set("opentracing-context", opentracing.ContextWithSpan(c, span))
		}

		// 响应日志封装
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()

		lg := &httpReqResLog{
			Operator:   getRequestUser(c.Request.Header),
			URI:        c.Request.URL.Path,
			Method:     c.Request.Method,
			Params:     string(params),
			Client:     c.ClientIP(),
			StatusCode: c.Writer.Status(),
		}
		if logResponse {
			lg.Response = blw.body.String()
		}

		if span != nil {
			span.LogFields(
				log.String("uri", lg.URI),
				log.String("method", lg.Method),
				log.String("params", lg.Params),
				log.Int("status_code", lg.StatusCode),
			)
		}

		logBytes, _ := json.Marshal(&lg)
		logger.Debugf("request details: %s", string(logBytes))
	}
}

func getRequestUser(header http.Header) string {
	if re, ok := header["X-Forwarded-User"]; ok {
		return re[0]
	}
	if re, ok := header["Authorization"]; ok && len(re) > 0 {
		// 简单脱敏处理 token
		return strings.Split(re[0], " ")[0] + ":***"
	}
	return ""
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, _user")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func GinFormatterLog() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(params gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s %d \"%s\" \"%s\" \"\n",
			params.ClientIP,
			params.TimeStamp.Format(time.RFC1123),
			params.Method,
			params.Path,
			params.Request.Proto,
			params.StatusCode,
			params.Latency,
			params.BodySize,
			params.Request.UserAgent(),
			params.ErrorMessage,
		)
	})
}

func AdminAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		expected := "AdminSecret " + secret
		if header != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized admin access",
			})
			return
		}
		c.Next()
	}
}

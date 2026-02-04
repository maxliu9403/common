package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"

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

// generateRequestID 生成唯一的请求 ID
// 格式: req_{timestamp_hex}_{random_hex}
// 例如: req_19c229a29e5_b8ec0e94
func generateRequestID() string {
	// 使用时间戳（毫秒）+ 随机数生成唯一 ID
	timestamp := time.Now().UnixMilli()
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	return fmt.Sprintf("req_%x_%s", timestamp, hex.EncodeToString(randomBytes))
}

// GinInterceptor HTTP 请求拦截器中间件
// 功能：记录请求日志、注入 request_id 实现全链路追踪
// 参数 logResponse: 是否记录响应体内容
func GinInterceptor(logResponse bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			bodyBytes []byte
			params    []byte
		)

		// ========== 1. 生成或提取 Request ID ==========
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		// 设置到 Gin Context（供 handler 层使用）
		c.Set("request_id", requestID)
		// 设置到 Response Header（供前端和下游服务使用）
		c.Header("X-Request-ID", requestID)
		// 注入到 Go Context（供 logger 自动提取）
		ctx := c.Request.Context()
		ctx = logger.WithRequestID(ctx, requestID)
		c.Request = c.Request.WithContext(ctx)

		// ========== 2. 读取请求体 ==========
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

		// ========== 3. 封装响应日志 Writer ==========
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()

		// ========== 4. 记录请求日志（结构化格式） ==========
		responseBody := ""
		if logResponse {
			responseBody = blw.body.String()
		}

		// 使用结构化日志，自动包含 request_id
		logger.InfowWithTrace(c.Request.Context(), "request details",
			"operator", getRequestUser(c.Request.Header),
			"uri", c.Request.RequestURI,
			"method", c.Request.Method,
			"params", string(params),
			"client", c.ClientIP(),
			"status_code", c.Writer.Status(),
			"response", responseBody,
		)
	}
}

// GinInterceptorWithTrace HTTP 请求拦截器中间件（带 OpenTracing 支持）
// 功能：记录请求日志、注入 request_id 和 tracing span 实现全链路追踪
// 参数 tra: OpenTracing tracer 实例
// 参数 logResponse: 是否记录响应体内容
func GinInterceptorWithTrace(tra opentracing.Tracer, logResponse bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			bodyBytes []byte
			err       error
			params    []byte
		)

		// ========== 1. 生成或提取 Request ID ==========
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		// 设置到 Gin Context（供 handler 层使用）
		c.Set("request_id", requestID)
		// 设置到 Response Header（供前端和下游服务使用）
		c.Header("X-Request-ID", requestID)
		// 注入到 Go Context（供 logger 自动提取）
		ctx := c.Request.Context()
		ctx = logger.WithRequestID(ctx, requestID)
		c.Request = c.Request.WithContext(ctx)

		// ========== 2. 读取请求体 ==========
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

		// ========== 3. OpenTracing Span 创建 ==========
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

			// 将 span 存入 context，同时保留 request_id
			spanContext := opentracing.ContextWithSpan(c.Request.Context(), span)
			c.Set("opentracing-context", spanContext)
			c.Request = c.Request.WithContext(spanContext)
		}

		// ========== 4. 封装响应日志 Writer ==========
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()

		// ========== 5. 记录到 Span ==========
		if span != nil {
			span.LogFields(
				log.String("request_id", requestID),
				log.String("uri", c.Request.URL.Path),
				log.String("method", c.Request.Method),
				log.String("params", string(params)),
				log.Int("status_code", c.Writer.Status()),
			)
		}

		// ========== 6. 记录请求日志（结构化格式） ==========
		responseBody := ""
		if logResponse {
			responseBody = blw.body.String()
		}

		// 使用结构化日志，自动包含 request_id、trace_id、span_id
		logger.DebugwWithTrace(c.Request.Context(), "request details",
			"operator", getRequestUser(c.Request.Header),
			"uri", c.Request.URL.Path,
			"method", c.Request.Method,
			"params", string(params),
			"client", c.ClientIP(),
			"status_code", c.Writer.Status(),
			"response", responseBody,
		)
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

// Cors 跨域中间件
// 支持 X-Request-ID header 用于全链路追踪
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, _user, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type, X-Request-ID")
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

// Copyright (c) 2016-2019 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package logger

// This package wraps logger functionality that is being used
// in kraken providing seamless migration tooling if needed
// and hides out some initialization details

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/maxliu9403/common/gadget"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Context key type for request_id to avoid key collisions
type contextKey string

// RequestIDKey is the context key for storing request_id
const RequestIDKey contextKey = "request_id"

var (
	DefaultLog *DemoLog
)

type DemoLog struct {
	*zap.SugaredLogger
	config      *zap.Config
	logDir      string
	logBaseName string
}

// configure a default logger
func init() {
	defaultConfig = &Options{}
	defaultConfig.zapConfig = zap.NewProductionConfig()
	defaultConfig.zapConfig.Encoding = ZapEncodeConsole.String()
	defaultConfig.zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	defaultConfig.zapConfig.DisableStacktrace = true

	DefaultLog = &DemoLog{
		config: &defaultConfig.zapConfig,
	}
	ConfigureLogger(defaultConfig)
}

func ConfigureLogger(logOptions *Options) *DemoLog {
	if logOptions.CompareOptions() != defaultConfig.CompareOptions() {
		DefaultLog.config.Level = logOptions.Level.Level()
		DefaultLog.config.Development = logOptions.Development

		if logOptions.EnableTrace {
			defaultConfig.zapConfig.DisableStacktrace = false
		}
		if logOptions.Encoding != "" && logOptions.Encoding.IsValid() {
			DefaultLog.config.Encoding = logOptions.Encoding.String()
		}

		if logOptions.LogPath != "" {
			pwd, _ := os.Getwd()
			logPath := path.Join(pwd, logOptions.LogPath)
			DefaultLog.logDir = logPath

			if err := os.MkdirAll(DefaultLog.logDir, 0755); err != nil {
				panic(err)
			}
			if logOptions.LogName == "" {
				DefaultLog.logBaseName = "app.log"
			} else {
				DefaultLog.logBaseName = logOptions.LogName + ".log"
			}

			fullPath := path.Join(DefaultLog.logDir, DefaultLog.logBaseName)
			var rotatePath string
			if fullPath[0:1] == "/" {
				rotatePath = fmt.Sprintf("rotate:/%%2F%s", fullPath[1:])
			} else {
				rotatePath = fmt.Sprintf("rotate:/%s", fullPath)
			}

			DefaultLog.config.OutputPaths = append(DefaultLog.config.OutputPaths, rotatePath)
			DefaultLog.config.ErrorOutputPaths = append(DefaultLog.config.ErrorOutputPaths, rotatePath)

			logRotate := logRotationConfig{initLumberjackConf(logOptions)}

			_ = zap.RegisterSink("rotate", func(u *url.URL) (zap.Sink, error) {
				logRotate.Filename = u.Path[1:]
				return &logRotate, nil
			})
		}
	}

	// Skip this wrapper in a call stack.
	logger, err := DefaultLog.config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}

	DefaultLog.SugaredLogger = logger.Sugar()

	return DefaultLog
}

func Default() *zap.SugaredLogger {
	return DefaultLog.SugaredLogger
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...interface{}) {
	Default().Debug(args...)
}

// DebugWithTrace 调试日志（带追踪字段）
// 自动从 ctx 中提取 request_id、trace_id、span_id 添加到日志
func DebugWithTrace(ctx context.Context, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Debug(args...)
		return
	}

	l := With(spanData...)
	l.Debug(args...)
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...interface{}) {
	Default().Info(args...)
}

func InfoWithTrace(ctx context.Context, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Info(args...)
		return
	}

	l := With(spanData...)
	l.Info(args...)
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...interface{}) {
	Default().Warn(args...)
}

func WarnWithTrace(ctx context.Context, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Warn(args...)
		return
	}

	l := With(spanData...)
	l.Warn(args...)
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...interface{}) {
	Default().Error(args...)
}

func ErrorWithTrace(ctx context.Context, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Error(args...)
		return
	}

	l := With(spanData...)
	l.Error(args...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...interface{}) {
	Default().Panic(args...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the logger then panics
func DPanic(args ...interface{}) {
	Default().DPanic(args...)
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the logger then panics.
func DPanicf(template string, args ...interface{}) {
	Default().DPanicf(template, args...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	Default().Fatal(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	Default().Debugf(template, args...)
}

// DebugfWithTrace 格式化调试日志（带追踪字段）
// 自动从 ctx 中提取 request_id、trace_id、span_id 添加到日志
func DebugfWithTrace(ctx context.Context, template string, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Debugf(template, args...)
		return
	}

	l := With(spanData...)
	l.Debugf(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	Default().Infof(template, args...)
}

func InfofWithTrace(ctx context.Context, template string, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Infof(template, args...)
		return
	}

	l := With(spanData...)
	l.Infof(template, args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	Default().Warnf(template, args...)
}

func WarnfWithTrace(ctx context.Context, template string, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Warnf(template, args...)
		return
	}

	l := With(spanData...)
	l.Warnf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	Default().Errorf(template, args...)
}

func ErrorfWithTrace(ctx context.Context, template string, args ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Errorf(template, args...)
		return
	}

	l := With(spanData...)
	l.Errorf(template, args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	Default().Panicf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Default().Fatalf(template, args...)
}

// Debugw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//
//	s.With(keysAndValues).Debug(msg)
func Debugw(msg string, keysAndValues ...interface{}) {
	Default().Debugw(msg, keysAndValues...)
}

// Infow logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Infow(msg string, keysAndValues ...interface{}) {
	Default().Infow(msg, keysAndValues...)
}

// Warnw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Warnw(msg string, keysAndValues ...interface{}) {
	Default().Warnw(msg, keysAndValues...)
}

// Errorw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Errorw(msg string, keysAndValues ...interface{}) {
	Default().Errorw(msg, keysAndValues...)
}

// Panicw logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func Panicw(msg string, keysAndValues ...interface{}) {
	Default().Panicw(msg, keysAndValues...)
}

// Fatalw logs a message with some additional context, then calls os.Exit. The
// variadic key-value pairs are treated as they are in With.
func Fatalw(msg string, keysAndValues ...interface{}) {
	Default().Fatalw(msg, keysAndValues...)
}

// ============================================================================
// 结构化日志方法（带追踪字段）- *wWithTrace 系列
// 这些方法会自动从 Context 中提取 request_id、trace_id、span_id 并添加到日志中
// ============================================================================

// DebugwWithTrace 结构化调试日志（带追踪字段）
// 自动从 ctx 中提取 request_id、trace_id、span_id 添加到日志
func DebugwWithTrace(ctx context.Context, msg string, keysAndValues ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Debugw(msg, keysAndValues...)
		return
	}
	// 将追踪字段放在最前面，用户自定义字段在后面
	allFields := append(spanData, keysAndValues...)
	Default().Debugw(msg, allFields...)
}

// InfowWithTrace 结构化信息日志（带追踪字段）
// 自动从 ctx 中提取 request_id、trace_id、span_id 添加到日志
func InfowWithTrace(ctx context.Context, msg string, keysAndValues ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Infow(msg, keysAndValues...)
		return
	}
	allFields := append(spanData, keysAndValues...)
	Default().Infow(msg, allFields...)
}

// WarnwWithTrace 结构化警告日志（带追踪字段）
// 自动从 ctx 中提取 request_id、trace_id、span_id 添加到日志
func WarnwWithTrace(ctx context.Context, msg string, keysAndValues ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Warnw(msg, keysAndValues...)
		return
	}
	allFields := append(spanData, keysAndValues...)
	Default().Warnw(msg, allFields...)
}

// ErrorwWithTrace 结构化错误日志（带追踪字段）
// 自动从 ctx 中提取 request_id、trace_id、span_id 添加到日志
func ErrorwWithTrace(ctx context.Context, msg string, keysAndValues ...interface{}) {
	spanData := extractSpan(ctx)
	if spanData == nil {
		Default().Errorw(msg, keysAndValues...)
		return
	}
	allFields := append(spanData, keysAndValues...)
	Default().Errorw(msg, allFields...)
}

// Errort uses fmt.Sprintf to log a templated message.
func Errort(template string, args ...interface{}) error {
	Default().Errorf(template, args...)
	return fmt.Errorf(template, args...)
}

// JSON logs a struct data
func JSON(msg string, data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		Default().Debug(fmt.Sprintf("[LogJson][%s] Failed: %s", msg, err.Error()))
		return
	}
	Default().Debug(fmt.Sprintf("[%s] %s", msg, string(b)))
}

// With adds a variadic number of fields to the logging context.
// It accepts a mix of strongly-typed zapcore.Field objects and loosely-typed key-value pairs.
func With(args ...interface{}) *zap.SugaredLogger {
	return Default().With(args...)
}

// extractSpan 从 Context 中提取追踪信息，包括 request_id、trace_id 和 span_id
// 返回的是 key-value 对的切片，用于结构化日志输出
func extractSpan(ctx context.Context) []interface{} {
	var res []interface{}

	// 1. 提取 request_id (优先级最高，便于 ELK 搜索)
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok && id != "" {
			res = append(res, "request_id", id)
		}
	}

	// 2. 提取 trace_id 和 span_id (来自 OpenTracing/Jaeger)
	spanCtx, err := gadget.ExtractTraceSpan(ctx)
	if err == nil {
		span := opentracing.SpanFromContext(spanCtx)
		if span != nil {
			if jaegerCtx, ok := span.Context().(jaeger.SpanContext); ok {
				res = append(res,
					"trace_id", jaegerCtx.TraceID().String(),
					"span_id", jaegerCtx.SpanID().String(),
				)
			}
		}
	}

	// 如果没有任何追踪信息，返回 nil
	if len(res) == 0 {
		return nil
	}

	return res
}

// WithRequestID 将 request_id 注入到 Context 中
// 用于在请求入口处（如中间件）设置 request_id，后续的日志调用会自动包含该 ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// RequestIDFromContext 从 Context 中提取 request_id
// 如果 Context 中不存在 request_id 或类型不匹配，返回空字符串
func RequestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

type Logger interface {
	// Error logs a message at error priority
	Error(msg string)

	// Infof logs a message at info priority
	Infof(msg string, args ...interface{})
}

func (d *DemoLog) Error(msg string) {
	Default().Error(msg)
}

func (d *DemoLog) Infof(template string, args ...interface{}) {
	Default().Infof(template, args...)
}

// PrintPrettyStructLog 打印结构体内容（格式化 JSON），主要用于调试或结构化输出。
func PrintPrettyStructLog(data interface{}) []byte {
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// 打印错误日志
		Default().Warnf("序列化失败: %v，使用原始格式", err)
		// 返回原始结构体的字符串表示（非 JSON）
		return []byte(fmt.Sprintf("%+v", data))
	}
	return pretty
}

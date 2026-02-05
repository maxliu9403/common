# Go 常用基础包

![common](./logo.png)

集成了项目开发过程中经常用到的基础包，欢迎 Merge Request。由于工具包的项目特性，所以代码平铺在根目录下。

## 安装

```bash
go get -u github.com/maxliu9403/common
```

---

## 目录结构

```
.
├── apiserver      # Gin Server 启停管理
├── cronjob        # 定时任务
├── etcd           # etcd 客户端封装
├── gadget         # 常用小工具（UUID、结构体处理等）
├── ginpprof       # Gin pprof 性能分析
├── gormdb         # GORM 封装（通用 CRUD、分页、模糊搜索）
├── httputil       # HTTP 客户端封装
├── kafka          # Kafka 生产者/消费者封装
├── logger         # 日志封装（Zap + 日志切割 + 全链路追踪）
├── middleware     # Gin 中间件（请求拦截、跨域、OpenTracing）
├── ratelimiter    # 限流器
├── rediscache     # Redis 客户端封装（分布式锁）
├── rsql           # RSQL 语法解析
├── tracer         # OpenTracing/Jaeger 封装
└── version        # 版本信息命令
```

---

## 核心功能

### Logger 日志

基于 Zap 封装的高性能日志库，支持：

- ✅ 日志切割（按大小/时间）
- ✅ 多输出格式（Console/JSON）
- ✅ 全链路追踪（自动注入 `request_id`、`trace_id`、`span_id`）
- ✅ Jaeger Span 日志（日志同时写入 Jaeger UI）

```go
import "github.com/maxliu9403/common/logger"

// 基础日志
logger.Info("message")
logger.Infof("user %s logged in", username)
logger.Infow("request received", "method", "GET", "path", "/api")

// 带追踪的日志（自动提取 context 中的 trace 信息）
logger.InfoWithTrace(ctx, "message")
logger.InfofWithTrace(ctx, "user %s logged in", username)
logger.InfowWithTrace(ctx, "request received", "method", "GET", "path", "/api")

// 所有 *WithTrace 函数都会同时写入到 Jaeger Span
// 支持的级别: Debug, Info, Warn, Error
```

**日志配置示例：**

```yaml
logger:
  level: info           # debug, info, warn, error
  encoding: json        # console, json
  log_path: ./logs      # 日志文件目录
  log_name: app         # 日志文件名
  max_size: 100         # 单文件最大 MB
  max_backups: 10       # 保留文件数
  max_age: 30           # 保留天数
  compress: true        # 是否压缩
```

---

### Middleware 中间件

#### GinInterceptorWithTrace

带 OpenTracing 支持的 HTTP 请求拦截器，功能：

- ✅ 自动生成/提取 `Request-ID`
- ✅ 创建 OpenTracing Span
- ✅ 记录请求/响应日志到 Jaeger
- ✅ 错误响应自动标记（4xx/5xx 高亮）
- ✅ 支持自定义 Span Tags

```go
import "github.com/maxliu9403/common/middleware"

// 基础用法
r.Use(middleware.GinInterceptorWithTrace(tracer, true))

// 自定义 Span Tags
r.Use(middleware.GinInterceptorWithTrace(tracer, true,
    middleware.WithTagProvider(func(c *gin.Context) map[string]interface{} {
        return map[string]interface{}{
            "user_id":   c.GetHeader("X-User-ID"),
            "tenant_id": c.GetHeader("X-Tenant-ID"),
            "app_name":  "my-service",
        }
    }),
))

// 多个 Provider 组合
r.Use(middleware.GinInterceptorWithTrace(tracer, true,
    middleware.WithTagProvider(userTagProvider),
    middleware.WithTagProvider(businessTagProvider),
))
```

**SpanTagProvider 函数签名：**

```go
type SpanTagProvider func(c *gin.Context) map[string]interface{}
```

#### 其他中间件

```go
// 跨域中间件（支持 X-Request-ID）
r.Use(middleware.Cors())

// 请求拦截器（不带 OpenTracing）
r.Use(middleware.GinInterceptor(true))

// 日志格式化
r.Use(middleware.GinFormatterLog())

// Admin 鉴权
r.Use(middleware.AdminAuthMiddleware("secret-key"))
```

---

### Tracer 链路追踪

基于 Jaeger 的 OpenTracing 实现：

```go
import "github.com/maxliu9403/common/tracer"

// 初始化 Jaeger Tracer
tra, closer, err := tracer.NewJaegerTracer("my-service", &tracer.Config{
    LocalAgentHostPort:  "jaeger-agent:6831",
    LogSpan:             true,
    BufferFlushInterval: 1,
}, logger.DefaultLog)
defer closer.Close()

// 配合中间件使用
r.Use(middleware.GinInterceptorWithTrace(tra, true))
```

---

### GormDB 数据库

GORM 封装，提供通用 CRUD 接口：

```go
import "github.com/maxliu9403/common/gormdb"

// 初始化
db := gormdb.NewDB(&gormdb.Config{...})
crud := gormdb.NewCRUD(db)

// CRUD 操作
crud.Create(&user)
crud.GetByID(&user, 1)
crud.GetOneByCon("name = ?", &user, "john")
crud.UpdateWithMap(&user, map[string]interface{}{"status": 1})
crud.Delete(&user, false) // false: 软删除, true: 硬删除

// 分页查询
total, err := crud.GetList(gormdb.BasicQuery{
    Limit:   10,
    Offset:  0,
    Keyword: "search",
    Order:   "created_at desc",
    Query:   "status==1",  // RSQL 语法
}, &User{}, &users)
```

---

### RedisCache 缓存

Redis 客户端封装，支持分布式锁：

```go
import "github.com/maxliu9403/common/rediscache"

// 初始化
cli := rediscache.NewClient(ctx, &rediscache.Config{...})
crud := rediscache.NewCRUD(ctx, cli)

// 基础操作
crud.Set("key", "value", time.Hour)
val, err := crud.Get("key")

// 分布式锁（非阻塞）
ok, err := crud.TryLock("lock-key", "unique-uuid", 30)
defer crud.UnLock("lock-key", "unique-uuid")

// 分布式锁（阻塞，带重试）
ok, err := crud.TryLockBlocking("lock-key", "uuid", 30, 3, 10*time.Second)
```

---

### Kafka 消息队列

异步生产者和消费者封装：

```go
import "github.com/maxliu9403/common/kafka"

// 生产者
producer := kafka.NewAsyncProducer(ctx, &kafka.Config{...})
producer.RunAsyncProducer()
producer.Produce("topic", []byte("message"))
defer producer.CloseProducer()

// 消费者
consumer := kafka.NewConsumer(ctx, &kafka.Config{...})
consumer.RunConsumer("group-id", []string{"topic"}, kafka.NewConsumerGroup(func(msg *sarama.ConsumerMessage) {
    // 处理消息
}))
defer consumer.Close()
```

---

### Gadget 工具包

常用小工具函数：

```go
import "github.com/maxliu9403/common/gadget"

// 生成 UUID
uuid := gadget.GetUUID()

// 结构体字段处理
fields := gadget.FieldsFromModel(&model, db, true)
columnMap := gadget.JsonTagColumnMapFromModel(&model, db)

// 提取 Trace Span
spanCtx, err := gadget.ExtractTraceSpan(ctx)
```

---

### APIServer 服务

Gin Server 生命周期管理：

```go
import "github.com/maxliu9403/common/apiserver"

server := apiserver.NewServer(&apiserver.Config{
    Port:         8080,
    ReadTimeout:  30,
    WriteTimeout: 30,
})

// 启动服务（阻塞，优雅关闭）
server.Run(router)
```

---

## 全链路追踪架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  Gin + Mid  │────▶│   Handler   │
│             │     │  (Tracer)   │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
                           │                   │
                           ▼                   ▼
                    ┌─────────────┐     ┌─────────────┐
                    │   Jaeger    │     │   Logger    │
                    │   (Spans)   │◀────│ (WithTrace) │
                    └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Jaeger UI  │
                    │  (Traces)   │
                    └─────────────┘
```

**日志与追踪关联：**

- `request_id`: 唯一请求标识，贯穿整个请求生命周期
- `trace_id`: Jaeger Trace ID，用于关联分布式服务
- `span_id`: Jaeger Span ID，标识当前操作

---

## License

Apache License 2.0

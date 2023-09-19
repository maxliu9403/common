# Go 常用基础包

![common](./logo.png)


集成了项目开发过程中经常用到的基础包，欢迎 Merge Request。由于工具包的项目特性，所以代码平铺在根目录下。

`go get -u github.com/maxliu9403/common`


## 简要说明

```
.
├── apiserver
│ 一组 gin server 启停的代码。实际使用案例可见 go-template 项目
├── cronjob
│ 定时任务
├── etcd
│ 封装的 etcd 客户端 
├── gadget
│ 一些常用的小函数，包含生成 UUID 等
├── ginpprof
│ 为 gin 提供 pprof
├── gormdb
│ gorm 封装，包含了常用的增删改查，具体实现见 repo.go
├── httputil
│ 封装的 http 客户端 
├── kafka
│ 封装的 Kafka 生产和消费客户端
├── logger
│ 封装的日志，支持切割
├── rediscache
│ 封装的 redis 客户端
├── rsql
│ rsql 语法解析，一般用不到
├── tracer
│ opentracing 封装
├── version
│ 为项目提供版本显示的命令

```

一些客户端使用方法可以参考对应的 `example`
/*
@Date: 2021/11/10 10:55
@Author: max.liu
@File : server
*/

package apiserver

import (
	"context"
	"io"
	"net/http"

	"github.com/maxliu9403/common/ginpprof"
	"github.com/maxliu9403/common/logger"
	"github.com/maxliu9403/common/middleware"
	"github.com/maxliu9403/common/tracer"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

const docJSON = "/swagger/doc.json"

type Server struct {
	conf        APIConfig
	logger      *logger.DemoLog
	adminEngine *gin.Engine
	engine      *gin.Engine
	tracer      opentracing.Tracer
	traceIO     io.Closer
}

// CreateNewServer create a new server with gin
func CreateNewServer(ctx context.Context, c APIConfig, opts ...ServerOption) *Server {
	server, err := newServer(ctx, c, opts)
	if err != nil {
		logger.Fatal(err)
	}

	return server
}

func newServer(ctx context.Context, c APIConfig, options []ServerOption) (server *Server, err error) {
	opts := &serverOptions{}
	for _, o := range options {
		o(opts)
	}

	server = &Server{
		conf:   c,
		logger: c.buildLogger(),
	}

	server.initGin()
	server.initAdmin()

	// tracer 初始化必须在其他组件之前
	if c.Tracer.LocalAgentHostPort != "" {
		tra, cli, e := tracer.NewJaegerTracer(c.App.ServiceName, &c.Tracer, server.logger)
		if e != nil {
			err = e
			return
		}

		server.tracer = tra
		server.traceIO = cli
	}

	return server, c.initService(ctx, opts)
}

func (s *Server) initGin() {
	switch s.conf.App.RunMode {
	case RunModeRelease, RunModeProd, RunModeProduction:
		gin.SetMode(gin.ReleaseMode)
	case RunModeTest, RunModeDev:
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	g := gin.New()
	// 开启跨域
	if s.conf.App.Cors == "1" {
		g.Use(gin.Recovery(), middleware.GinFormatterLog(), middleware.Cors())
	} else {
		g.Use(gin.Recovery(), middleware.GinFormatterLog())
	}

	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"RetCode": 0,
			"Message": "pong",
		})
	})

	if s.conf.App.RunMode == RunModeDebug || s.conf.App.RunMode == RunModeRelease || s.conf.App.RunMode == RunModeDev {
		g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		// 要求文档统一入口为 api-docs，重定向请求
		g.GET("/api-docs", func(c *gin.Context) {
			c.Request.URL.Path = "/swagger/index.html"
			c.Request.RequestURI = "/swagger/index.html"
			g.HandleContext(c)
		})
		g.GET("/api-docs.json", func(c *gin.Context) {
			c.Request.URL.Path = docJSON
			c.Request.RequestURI = docJSON
			g.HandleContext(c)
		})
		g.GET("/doc.json", func(c *gin.Context) {
			c.Request.URL.Path = docJSON
			c.Request.RequestURI = docJSON
			g.HandleContext(c)
		})
		g.GET("/swagger-ui.css", func(c *gin.Context) {
			c.Request.URL.Path = "/swagger/swagger-ui.css"
			c.Request.RequestURI = "/swagger/swagger-ui.css"
			g.HandleContext(c)
		})
		g.GET("/swagger-ui-bundle.js", func(c *gin.Context) {
			c.Request.URL.Path = "/swagger/swagger-ui-bundle.js"
			c.Request.RequestURI = "/swagger/swagger-ui-bundle.js"
			g.HandleContext(c)
		})
		g.GET("/swagger-ui-standalone-preset.js", func(c *gin.Context) {
			c.Request.URL.Path = "/swagger/swagger-ui-standalone-preset.js"
			c.Request.RequestURI = "/swagger/swagger-ui-standalone-preset.js"
			g.HandleContext(c)
		})
	} else {
		gin.DisableConsoleColor()
	}

	s.engine = g
}

func (s *Server) initAdmin() {
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	g := gin.New()
	g.Use(middleware.GinFormatterLog(), gin.Recovery())

	ginpprof.Wrap(g)
	logger.Wrap(g)

	s.adminEngine = g
}

func (s *Server) AddGinGroup(group string) *gin.RouterGroup {
	return s.engine.Group(group)
}

// 对外暴露服务的 gin.Engine, 仅推荐写接口的单元测试时使用
func (s *Server) ExposeEng() *gin.Engine {
	return s.engine
}

func (s *Server) GetTracer() opentracing.Tracer {
	return s.tracer
}

func (s *Server) Start() {
	s.logger.Infof("starting server at %s: %d", s.conf.App.HostIP, s.conf.App.APIPort)

	go func() {
		s.logger.Infof("starting admin server at %s: %d", s.conf.App.HostIP, s.conf.App.AdminPort)
		err := StartHTTP(s.conf.App.HostIP, s.conf.App.AdminPort, s.adminEngine)
		handleError(err)
	}()

	var err error
	if s.conf.App.CertFile != "" && s.conf.App.KeyFile != "" {
		err = StartHTTPS(s.conf, s.engine)
	} else {
		err = StartHTTP(s.conf.App.HostIP, s.conf.App.APIPort, s.engine)
	}

	handleError(err)
}

func (s *Server) StartAdminOnly() {
	s.logger.Infof("starting admin server at %s: %d", s.conf.App.HostIP, s.conf.App.AdminPort)
	err := StartHTTP(s.conf.App.HostIP, s.conf.App.AdminPort, s.adminEngine)

	handleError(err)
}

func (s *Server) Stop() {
	_ = s.logger.Sync()

	if s.tracer != nil {
		_ = s.traceIO.Close()
	}
}

func handleError(err error) {
	// ErrServerClosed means the server is closed manually
	if err == nil || err == http.ErrServerClosed {
		return
	}

	logger.Fatal(err)
}

/*
@Date: 2021/10/26 11:50
@Author: max.liu
@File : main
*/

package main

import (
	"github.com/common/logger"
	"github.com/common/middleware"
	"github.com/common/tracer"
	"github.com/gin-gonic/gin"
)

func main() {
	c := tracer.Config{
		BufferFlushInterval: 10,
		LocalAgentHostPort:  "127.0.0.1:6831",
		LogSpan:             true,
	}
	initTrace, cl, err := tracer.NewJaegerTracer("gin-example", &c, logger.DefaultLog)
	if err != nil {
		panic(err)
	}
	defer cl.Close()

	r := gin.New()
	r.Use(middleware.GinInterceptorWithTrace(initTrace, false))

	r.POST("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run()
}

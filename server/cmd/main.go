package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/config"
	"github.com/gwyy/WebTrailAI/server/internal/ctrl"
	"github.com/gwyy/WebTrailAI/server/internal/logger"
	"github.com/gwyy/WebTrailAI/server/internal/router"
	"github.com/gwyy/WebTrailAI/server/internal/service"
)

func main() {
	//获取配置文件
	var configFile string
	flag.StringVar(&configFile, "conf", "./config.yaml", "config file path")
	flag.Parse()

	//Init App
	newConfig := config.NewConfig(configFile)
	newLogger := logger.NewLogger(newConfig)
	newService := service.NewService(newConfig, newLogger)
	newCtrl := ctrl.NewCtrl(newService, newConfig, newLogger)
	newRouter := router.NewRouter(newCtrl, newConfig, newLogger)

	engine := gin.New()        //gin 引擎
	engine.Use(gin.Recovery()) //添加 recovery 中间件
	//设置gin模式
	gin.SetMode(getGinMode(newConfig.GetString("mode")))
	if newConfig.GetString("mode") == "dev" || newConfig.GetString("mode") == "test" {
		engine.Use(gin.Logger())
	}
	//初始化 router
	newRouter.Init(engine)

	// 获取端口
	address := fmt.Sprintf(":%d", newConfig.GetInt("gin.port"))
	//初始化 http 服务器
	server := http.Server{
		Addr:    address,
		Handler: engine,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	newLogger.Infof(`
	%s 启动成功
	当前版本:v%s
	默认前端文件运行地址:http://127.0.0.1%s
`, newConfig.GetString("name"), newConfig.GetString("version"), address)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	//在此阻塞
	<-quit
	//等待 5 秒
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		if err == context.DeadlineExceeded {
			newLogger.Info("Shutdown DeadlineExceeded")
		} else {
			newLogger.Fatal("server shutdown error: " + err.Error())
		}
	} else {
		newLogger.Info("server shutdown gracefully")
	}
}

// 获取gin模式
func getGinMode(mode string) string {
	if mode == "dev" {
		return gin.DebugMode
	} else if mode == "test" {
		return gin.TestMode
	} else {
		return gin.ReleaseMode
	}
}

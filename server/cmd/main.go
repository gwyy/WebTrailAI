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
	scribble_manager "github.com/gwyy/WebTrailAI/server/pkg/scribble-manager"
	"github.com/gwyy/WebTrailAI/server/pkg/utils"
)

func main() {
	//获取配置文件
	var configFile string
	flag.StringVar(&configFile, "conf", "./config.yaml", "config file path")
	flag.Parse()

	//Init App
	newConfig := config.NewConfig(configFile)
	newLogger := logger.NewLogger(newConfig)
	//实例化 model + service
	newSm, err := scribble_manager.NewScribbleManager(newConfig, newLogger)
	if err != nil {
		newLogger.Fatal(err)
	}
	//实例化 service
	newService := service.NewService(newConfig, newLogger, newSm)

	//实例化 阿里云 大模型接口
	if llmClient, err := utils.NewDashScopeClient(utils.LLMOptions{
		APIKey:  newConfig.GetString("ai.dashscope_api_key"),
		BaseURL: newConfig.GetString("ai.base-url"),
		Model:   newConfig.GetString("ai.model"),
		Timeout: time.Duration(newConfig.GetInt("ai.timeout-seconds")) * time.Second,
	}); err != nil {
		newLogger.Warnf("大模型客户端初始化失败，每日浏览总结任务将不可用: %v", err)
	} else {
		newService.SetLLMClient(llmClient)
	}
	//实例化 controller
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
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	newLogger.Infof(`
	%s 启动成功
	当前版本:v%s
	默认前端文件运行地址:http://127.0.0.1%s
`, newConfig.GetString("name"), newConfig.GetString("version"), address)

	//实现 crontab
	var summaryScheduler *service.SummaryScheduler
	if newConfig.GetBool("summary.enabled") {
		summaryScheduler = service.NewSummaryScheduler(newService)
		if err = summaryScheduler.Start(context.Background(), newConfig.GetString("summary.cron")); err != nil {
			newLogger.Errorf("每日浏览总结任务启动失败: %v", err)
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	//在此阻塞
	<-quit
	if summaryScheduler != nil {
		summaryScheduler.Stop()
	}
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

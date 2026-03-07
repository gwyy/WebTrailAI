package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/config"
	"github.com/gwyy/WebTrailAI/server/ctrl"
	"github.com/gwyy/WebTrailAI/server/service"
	"github.com/spf13/viper"
)

// App 结构体
type App struct {
	route   *Router
	ctrl    *ctrl.Ctrl
	cfg     config.Config
	viper   *viper.Viper
	service service.Service
	engine  *gin.Engine
}

var GlobalApp *App

// 全局 App
func NewApp() *App {
	app := &App{
		//route:  route,
		engine: gin.Default(),
	}
	//app.route.Init(app.engine)
	return app
}

// 初始化 App
func InitApp(configFile string) *App {
	GlobalApp = NewApp()
	GlobalApp.viper = config.NewConfig(configFile)
	newService := service.NewService(newConfig)
	newCtrl := ctrl.NewCtrl(newService, newConfig)
	router := NewRouter(newCtrl, newConfig)
	app := NewApp(router)
	return app
}

func main() {
	//获取配置文件
	var configFile string
	flag.StringVar(&configFile, "conf", "./config.yaml", "config file path")
	flag.Parse()

	InitApp(configFile)
	server := http.Server{
		Addr:    ":8080",
		Handler: app.engine,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	//在此阻塞
	<-quit
	//等待 5 秒
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		if err == context.DeadlineExceeded {
			log.Println("Shutdown DeadlineExceeded")
		} else {
			log.Fatal("server shutdown error: " + err.Error())
		}
	} else {
		log.Println("server shutdown gracefully")
	}
}

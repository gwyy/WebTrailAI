package router

import (
	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/config"
	"github.com/gwyy/WebTrailAI/server/internal/ctrl"
	"github.com/gwyy/WebTrailAI/server/internal/logger"
	"github.com/gwyy/WebTrailAI/server/internal/model/response"
)

type Router struct {
	ctrl *ctrl.Ctrl
	cfg  config.Config
	log  logger.Logger
}

func NewRouter(ctrl *ctrl.Ctrl, cfg config.Config, log logger.Logger) *Router {
	return &Router{ctrl: ctrl, cfg: cfg, log: log}
}

// 初始化 router
func (r *Router) Init(gin *gin.Engine) {

	gin.GET("/", r.ctrl.Hello)
	//空 route
	gin.NoRoute(handleNoRoute())

	//实例化 jwt
	authJwtMiddleware, err := jwt.New(r.initJWTParams(r.ctrl.Authenticator))
	if err != nil || authJwtMiddleware == nil {
		r.log.Fatal("JWT Error:" + err.Error())
	}
	// initialize middleware
	errInit := authJwtMiddleware.MiddlewareInit()
	if errInit != nil {
		r.log.Fatal("authMiddleware.MiddlewareInit() Error:" + errInit.Error())
	}
	// ==================== jwt 公开路由（不需要 Token）===================
	gin.POST("/login", authJwtMiddleware.LoginHandler)
	gin.POST("/refresh_token", authJwtMiddleware.RefreshHandler) // RFC 6749 compliant refresh endpoint
	gin.POST("/logout", authJwtMiddleware.LogoutHandler)         // 登出（会调用你写的 logoutResponse）

	// ==================== 其他 公开路由（不需要 Token）===================
	gin.POST("/register", r.ctrl.Register)
	// ==================== 保护路由组（需要 Token）===================
	protected := gin.Group("/api")                    // 你可以改成 /admin、/user 等
	protected.Use(authJwtMiddleware.MiddlewareFunc()) // ← 关键！所有下面接口都要登录
	{
		protected.GET("/list", r.ctrl.TrailList)
	}
}

func handleNoRoute() func(c *gin.Context) {
	return func(c *gin.Context) {
		response.FailWithCodeAndMessage(response.NotFound, "Not Found", c)
	}
}

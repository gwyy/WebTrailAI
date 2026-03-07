package ctrl

import (
	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/internal/config"
	"github.com/gwyy/WebTrailAI/server/internal/logger"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
	"github.com/gwyy/WebTrailAI/server/internal/service"
)

type Ctrl struct {
	srv *service.Service
	cfg config.Config
	log logger.Logger
}

func NewCtrl(srv *service.Service, cfg config.Config, log logger.Logger) *Ctrl {
	return &Ctrl{
		srv: srv,
		cfg: cfg,
		log: log,
	}
}

func (r *Ctrl) Hello(c *gin.Context) {
	c.JSON(200, gin.H{"hello": "world"})
}

func (r *Ctrl) parseUser(c *gin.Context) *request.JwtUser {
	if JwtUserStr, exists := c.Get("id"); exists {
		if JwtUser, ok := JwtUserStr.(*request.JwtUser); ok {
			return JwtUser
		}
	}
	return nil
}

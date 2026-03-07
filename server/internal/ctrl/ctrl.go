package ctrl

import (
	"github.com/gin-gonic/gin"
	"github.com/gwyy/WebTrailAI/server/config"
	"github.com/gwyy/WebTrailAI/server/service"
)

type Ctrl struct {
	srv *service.Service
	cfg *config.Config
}

func NewCtrl(srv *service.Service, cfg *config.Config) *Ctrl {
	return &Ctrl{
		srv: srv,
		cfg: cfg,
	}
}

func (r *Ctrl) Hello(c *gin.Context) {
	c.JSON(200, gin.H{"hello": "world"})
}

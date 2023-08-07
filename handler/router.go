package handler

import (
	"goplay/config"

	"github.com/gin-gonic/gin"
)

func StartRouter(cfg *config.Config, handler *HttpHandler) {
	r := gin.Default()

	r.POST("/teams", handler.AddGroup)
	r.DELETE("/teams", handler.RemoveGroup)
	r.POST("/players/ready", handler.SetPlayerReady)

	r.Run(cfg.Server.Port)
}

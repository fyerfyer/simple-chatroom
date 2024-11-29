package routers

import (
	"github.com/fyerfyer/chatroom/models"
	"github.com/fyerfyer/chatroom/routers/api"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	go models.Broadcaster.Start()
	r.GET("/user_list", api.UserListHandler)
	r.GET("/ws", api.WebSocketHandler)

	return r
}

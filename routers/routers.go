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
	r.GET("/users", func(c *gin.Context) {
		users := models.Broadcaster.GetUserList()
		c.JSON(200, users)
	})

	return r
}

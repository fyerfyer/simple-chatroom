package api

import (
	"net/http"

	"github.com/fyerfyer/chatroom/models"
	"github.com/gin-gonic/gin"
)

func UserListHandler(c *gin.Context) {
	userList := models.Broadcaster.GetUserList()
	c.JSON(http.StatusOK, userList)
}

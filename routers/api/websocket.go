package api

import (
	"log"

	"github.com/fyerfyer/chatroom/models"
	"github.com/fyerfyer/chatroom/pkg/utils"
	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func WebSocketHandler(c *gin.Context) {
	conn, err := websocket.Accept(c.Writer, c.Request,
		&websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	// handling user login
	username := c.DefaultQuery("name", "")
	log.Println(username)
	if err := utils.ValidateName(username); err != nil {
		log.Printf("illegal username: %v", username)
		wsjson.Write(c.Request.Context(), conn, err)
		conn.Close(websocket.StatusUnsupportedData, "illegal username")
		return
	}

	// check if the user has login in
	if !models.Broadcaster.CheckUserCanLogin(username) {
		log.Printf("user already exists: %v", username)
		wsjson.Write(c.Request.Context(), conn, err)
		conn.Close(websocket.StatusUnsupportedData, "illegal username")
		return
	}

	// open up message-sending channel for user
	user := models.NewUser(conn, username, c.Request.RemoteAddr)
	go user.SendMessage(c)

	// send welcome message to current user
	user.MessageChannel <- models.NewWelcomeMsg(user)

	// inform all users that a new user has come
	msg := models.NewLoginMsg(user)
	models.Broadcaster.Broadcast(msg)

	// add the user to the userlist of broadcaster
	models.Broadcaster.UserLogin(user)
	log.Println(username, " has entered the chatroom")

	// fetcht the user's message
	err = user.FetchMessage(c)

	// the user is going to leave
	models.Broadcaster.UserLogout(user)
	msg = models.NewLogoutMsg(user)
	models.Broadcaster.Broadcast(msg)
	log.Println(username, "has exited the chatroom!")

	if err == nil {
		conn.Close(websocket.StatusNormalClosure, "")
	} else {
		log.Println("failed to fetch message: ", err)
		conn.Close(websocket.StatusInternalError, "Read from client error")
	}
}

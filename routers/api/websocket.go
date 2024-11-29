package api

import (
	"errors"
	"log"

	"github.com/fyerfyer/chatroom/models"
	"github.com/fyerfyer/chatroom/pkg/utils"
	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var (
	duplicateLoginErr  = errors.New("duplicate login")
	duplicateLogoutErr = errors.New("duplicate logout")
)

func handleError(c *gin.Context, conn *websocket.Conn,
	msg string, status websocket.StatusCode, statusMsg string) {
	if msg != "" {
		wsjson.Write(c, conn, msg)
	}

	conn.Close(status, statusMsg)
}

func WebSocketHandler(c *gin.Context) {
	conn, err := initWebSocketConnection(c)
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}

	user, err := authenticateUser(c, conn)
	if err != nil {
		handleError(c, conn, err.Error(),
			websocket.StatusUnsupportedData, "user login error")
		return
	}

	setupUserSession(c, user)

	if err := handleUserMessaging(c, user); err != nil {
		handleError(c, conn, err.Error(),
			websocket.StatusInternalError, "message handling error")
		return
	}

	if err := teardownUserSession(user); err != nil {
		handleError(c, conn, err.Error(),
			websocket.StatusInternalError, "user logout error")
		return
	}

	conn.Close(websocket.StatusNormalClosure, "")
}

func initWebSocketConnection(c *gin.Context) (*websocket.Conn, error) {
	conn, err := websocket.Accept(c.Writer, c.Request, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func authenticateUser(c *gin.Context, conn *websocket.Conn) (*models.User, error) {
	username := c.DefaultQuery("name", "")
	// log.Println("start authenticate user...")
	if err := utils.ValidateName(username); err != nil {
		log.Printf("illegal username: %v", username)
		return nil, err
	}

	if !models.Broadcaster.CheckUserCanLogin(username) {
		log.Printf("user already existed: %v", username)
		return nil, duplicateLoginErr
	}

	user := models.NewUser(conn, username, c.Request.RemoteAddr)
	log.Printf("user authenticated: %s", username)
	return user, nil
}

func setupUserSession(c *gin.Context, user *models.User) {
	// Start the message-sending goroutine.
	go user.SendMessage(c)

	// Send welcome message to the user.
	user.MessageChannel <- models.NewWelcomeMsg(user)

	// Add user to broadcaster's active user list.
	models.Broadcaster.UserLogin(user)

	log.Printf("%s has entered the chatroom", user.Name)
}

func handleUserMessaging(c *gin.Context, user *models.User) error {
	if err := user.FetchMessage(c); err != nil {
		log.Printf("failed to handle user messaging: %v", err)
		return err
	}

	return nil
}

func teardownUserSession(user *models.User) error {
	// Remove the user from the active list and broadcast the logout event.
	if !models.Broadcaster.CheckUserCanLogout(user.Name) {
		log.Printf("user already logout: %v", user.Name)
		return duplicateLogoutErr
	}

	models.Broadcaster.UserLogout(user)
	log.Printf("%s has exited the chatroom", user.Name)
	user.CloseChannel()
	return nil
}

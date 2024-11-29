package api

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fyerfyer/chatroom/models"
	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func init() {
	go models.Broadcaster.Start()
	time.Sleep(50 * time.Millisecond)
}

func TestHandleAuthenticateUser(t *testing.T) {
	welcomeMsg := "hi there, "

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Errorf("failed to accepted websocket connection: %v", err)
			return
		}

		defer conn.Close(websocket.StatusInternalError, "connection closed")

		user, err := authenticateUser(c, conn)
		if err != nil {
			wsjson.Write(c.Request.Context(), conn, "invalid user input")
			conn.Close(websocket.StatusUnsupportedData, err.Error())
			return
		}

		// if authenticated, write json welcome message
		wsjson.Write(c.Request.Context(), conn, map[string]string{
			"message": welcomeMsg + user.Name,
		})
	})

	server := httptest.NewServer(r)
	defer server.Close()

	t.Run("normal login", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		url := "ws://" + server.Listener.Addr().String() + "/ws?name=valid_user"
		conn, _, err := websocket.Dial(ctx, url, nil)
		if err != nil {
			t.Errorf("failed to establish websocket connection: %v", err)
			return
		}

		var res map[string]string
		err = wsjson.Read(ctx, conn, &res)
		if err != nil {
			t.Errorf("failed to get normal login response: %v", err)
			return
		}
		if res["message"] != welcomeMsg+"valid_user" {
			t.Errorf("should get login message %v, but got %v",
				welcomeMsg+"valid_user", res["message"])
			return
		}
		conn.Close(websocket.StatusNormalClosure, "")
	})

	t.Run("invalid username", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		url := "ws://" + server.Listener.Addr().String() + "/ws?name=u"
		conn, _, err := websocket.Dial(ctx, url, nil)
		if err != nil {
			t.Errorf("failed to establish websocket connection: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "test completed")

		// in out authenticate logic, when meeting error, error will be written into the conn
		// so we read the error
		var res interface{}
		err = wsjson.Read(ctx, conn, &res)
		if err != nil {
			t.Errorf("failed to get error response: %v", err)
			return
		}

		if res != "invalid user input" {
			t.Errorf("should pass 'invalid user input' error message but got: %v", res)
			return
		}
	})
}

func TestSetUpUserSession(t *testing.T) {
	user := &models.User{}
	var wantedMsg *models.Message

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Errorf("failed to accepted websocket connection: %v", err)
			return
		}

		defer conn.Close(websocket.StatusInternalError, "connection closed")

		user = models.NewUser(conn, "testing_user", "127.0.0.1")
		wantedMsg = models.NewWelcomeMsg(user)
		setupUserSession(c, user)
		wsjson.Write(c, conn, wantedMsg.Content)
	})

	server := httptest.NewServer(r)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws://" + server.Listener.Addr().String() + "/ws?name=" + user.Name
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Errorf("failed to establish websocket connection: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "test completed")

	// check message fetching
	var msg interface{}
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		t.Errorf("failed to get error response: %v", err)
		return
	}

	if msg != wantedMsg.Content {
		t.Errorf("wanted user get message %v but got: %v",
			msg, wantedMsg.Content)
		return
	}
}

func TestTearDownUserSession(t *testing.T) {
	var user = &models.User{}
	var wantedMsg = "user logout"
	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Errorf("failed to accepted websocket connection: %v", err)
			return
		}

		defer conn.Close(websocket.StatusInternalError, "connection closed")
		user = models.NewUser(conn, "testing_user", "127.0.0.1")

		teardownUserSession(user)
		wsjson.Write(c, conn, wantedMsg)
		err = teardownUserSession(user)
		if !errors.Is(err, duplicateLogoutErr) {
			t.Error("failed to get duplicate logout error")
			return
		}
	})

	server := httptest.NewServer(r)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws://" + server.Listener.Addr().String() + "/ws?name=" + user.Name
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Errorf("failed to establish websocket connection: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "test completed")

	var msg interface{}
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		t.Errorf("failed to get error response: %v", err)
		return
	}

	if msg != wantedMsg {
		t.Errorf("wanted user get messe %v but got: %v",
			msg, wantedMsg)
		return
	}
}

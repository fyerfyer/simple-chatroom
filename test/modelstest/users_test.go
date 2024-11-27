package modelstest

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/chatroom/models"
	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func init() {
	go models.StartTest()
	time.Sleep(50 * time.Millisecond)
}

func TestSendMessage(t *testing.T) {
	var wantedMsg = "Hello from server!"
	r := gin.Default()
	// setup websocket test router
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Fatalf("failed to accept websocket connection: %v", err)
		}

		defer conn.Close(websocket.StatusInternalError, "connection closed")

		user := models.NewUser(conn, "testing_user", "127.0.0.1")
		go func() {
			user.MessageChannel <- models.NewMessage(user, models.MsgTypeNormal, wantedMsg)
		}()

		time.Sleep(50 * time.Millisecond)
		user.SendMessage(c)
		close(user.MessageChannel)
	})

	// start server
	server := httptest.NewServer(r)
	defer server.Close()

	// create websocket client connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws://"+server.Listener.Addr().String()+"/ws", nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}

	defer conn.Close(websocket.StatusNormalClosure, "test connection closed")

	// fetch message
	var receivedMsg map[string]interface{}
	err = wsjson.Read(ctx, conn, &receivedMsg)
	if err != nil {
		t.Error("failed to received json message")
	}

	if msgContent := receivedMsg["content"]; msgContent != wantedMsg {
		t.Errorf("wanted message %v, but got %v", wantedMsg, msgContent)
	}
}

func TestFetchMessage(t *testing.T) {
	wantedMsg := "Hello from client!"
	user := &models.User{MessageChannel: make(chan *models.Message, 32)}

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Fatalf("failed to accept websocket connection: %v", err)
		}
		defer conn.Close(websocket.StatusInternalError, "connection closed")
		user = models.NewUser(conn, "testing_user", "127.0.0.1")
		user.FetchMessageForTesting(c)
	})

	server := httptest.NewServer(r)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws://"+server.Listener.Addr().String()+"/ws", nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test completed")

	models.LoginUserWithoutSendingMessage(user)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wsjson.Write(ctx, conn, models.NewMessage(user, models.MsgTypeNormal, wantedMsg))
	}()

	// Wait for the message to be sent before checking the channel
	wg.Wait()

	select {
	case msg := <-user.MessageChannel:
		if msg.Content != wantedMsg {
			t.Errorf("expected message: %v, got: %v", wantedMsg, msg.Content)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for message")
	}
}

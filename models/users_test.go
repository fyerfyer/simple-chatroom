package models

import (
	"context"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func init() {
	go Broadcaster.Start()
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

		user := NewUser(conn, "testing_user", "127.0.0.1")
		user.MessageChannel <- NewMessage(user, MsgTypeNormal, wantedMsg)
		user.SendMessage(c)
		close(user.MessageChannel)
	})

	// start server
	server := httptest.NewServer(r)
	defer server.Close()

	// create websocket client connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	url := "ws://" + server.Listener.Addr().String() + "/ws"
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}

	defer conn.Close(websocket.StatusNormalClosure, "test connection closed")

	// fetch message
	var receivedMsg map[string]interface{}
	err = wsjson.Read(ctx, conn, &receivedMsg)
	if err != nil {
		t.Error("failed to received json message")
		return
	}

	if msgContent := receivedMsg["content"]; msgContent != wantedMsg {
		t.Errorf("wanted message %v, but got %v", wantedMsg, msgContent)
		return
	}
}

func TestFetchMessage(t *testing.T) {
	wantedMsg := "Hello from client!"
	user := &User{}

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Fatalf("failed to accept websocket connection: %v", err)
		}
		defer conn.Close(websocket.StatusInternalError, "connection closed")
		user = NewUser(conn, "testing_user", "127.0.0.1")
		loginUserWithoutSendingMessage(user)
		user.FetchMessage(c)
	})

	server := httptest.NewServer(r)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws://" + server.Listener.Addr().String() + "/ws"
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test completed")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wsjson.Write(ctx, conn, NewMessage(user, MsgTypeNormal, wantedMsg))
	}()

	// Wait for the message to be sent before checking the channel
	wg.Wait()

	select {
	case msg := <-user.MessageChannel:
		if msg.Content != wantedMsg {
			t.Errorf("expected message: %v, got: %v", wantedMsg, msg.Content)
			return
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for message")
		return
	}
}

func TestUserMethodInteraction(t *testing.T) {
	user := &User{}
	wantedMsg := "Hello from testing_user!"

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Fatalf("failed to accept websocket connection: %v", err)
		}

		defer conn.Close(websocket.StatusInternalError, "connection closed")

		user = NewUser(conn, "testing_user", "127.0.0.1")

		// use usermethod to send and fetch message
		msg := NewMessage(user, MsgTypeNormal, wantedMsg)

		go func() {
			user.MessageChannel <- msg
			user.SendMessage(c)
		}()
		loginUserWithoutSendingMessage(user)
		user.FetchMessage(c)

		defer user.CloseChannel()
	})

	server := httptest.NewServer(r)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws://" + server.Listener.Addr().String() + "/ws"
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test completed")

	select {
	case msg := <-user.MessageChannel:
		if msg.Content != wantedMsg {
			t.Errorf("expected message: %v, got: %v", wantedMsg, msg.Content)
			return
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for message")
		return
	}
}

func TestUserConcurrency(t *testing.T) {
	user := &User{}
	sentMsg := "send message to"

	r := gin.Default()
	r.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, nil)
		if err != nil {
			t.Fatalf("failed to accept websocket connection: %v", err)
		}

		user = NewUser(conn, "testing_user", "127.0.0.1")
		loginUserWithoutSendingMessage(user)

		go func() {
			user.SendMessage(c)
		}()

		go func() {
			if err := user.FetchMessage(c); err != nil {
				t.Errorf("failed to fetch message: %v", err)
				return
			}
		}()
	})

	server := httptest.NewServer(r)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws://" + server.Listener.Addr().String() + "/ws"
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test completed")

	var wg sync.WaitGroup

	// write in concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			msg := NewMessage(user, MsgTypeNormal, sentMsg+strconv.Itoa(i))
			if err := wsjson.Write(ctx, conn, msg); err != nil {
				t.Errorf("Failed to write JSON data: %v", err)
				return
			}
		}(i)
	}
	wg.Wait()

	cnt := 0
	// fetch message concurrently
	go func() {
		for {
			select {
			case msg, ok := <-user.MessageChannel:
				if !ok {
					if cnt == 10 {
						t.Logf("successfully fetch all messages")
						return
					} else {
						t.Errorf("should fetched 10 messages, but got %v", cnt)
						return
					}
				} else {
					t.Logf("Processed message: %v", msg)
					cnt++
				}

			case <-time.After(2 * time.Second):
				t.Errorf("timeout waiting for message")
				return
			}
		}
	}()
}

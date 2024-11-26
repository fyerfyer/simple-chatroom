package models

import (
	"errors"
	"io"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/fyerfyer/chatroom/pkg/setting"
	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var globalUserID uint32 = 0
var System = &User{}

type User struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	CreatedAt      time.Time     `json:"created_at"`
	Addr           string        `json:"address"`
	MessageChannel chan *Message `json:"-"`

	conn     *websocket.Conn `json:"-"`
	IsOnline bool            `json:"-"`
}

func NewUser(conn *websocket.Conn, name, addr string) *User {
	user := &User{
		Name:           name,
		CreatedAt:      time.Now(),
		Addr:           addr,
		MessageChannel: make(chan *Message, setting.UserMessageQueueLength),
		conn:           conn,
	}

	if user.ID == 0 {
		// set user id if it haven't been set
		user.ID = int(atomic.AddUint32(&globalUserID, 1))
	}

	return user
}

func (u *User) SendMessage(c *gin.Context) {
	for msg := range u.MessageChannel {
		wsjson.Write(c, u.conn, msg)
	}
}

func (u *User) CloseChannel() {
	close(u.MessageChannel)
}

func (u *User) FetchMessage(c *gin.Context) error {
	var msg map[string]interface{}

	for {
		err := wsjson.Read(c, u.conn, &msg)
		if err != nil {
			var closeErr websocket.CloseError
			switch {
			case errors.As(err, &closeErr):
				return nil
			case errors.Is(err, io.EOF):
				return nil
			default:
				return err
			}
		}

		// send the message to the chatroom
		sendMsg := NewMessage(u, MsgTypeNormal, msg["content"].(string), msg["sent_at"].(time.Time))
		reg := regexp.MustCompile(`@[^\s@]{2,20}`)
		sendMsg.Ats = reg.FindAllString(sendMsg.Content, -1)

		// broadcast the message
		Broadcaster.Broadcast(sendMsg)
	}
}

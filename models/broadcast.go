package models

import (
	"log"

	"github.com/fyerfyer/chatroom/pkg/setting"
)

type broadcast struct {
	users map[string]*User
	ops   chan broadcastOp

	messageChannel chan *Message
}

type broadcastOp struct {
	typ   string
	user  *User
	reply chan interface{}
}

const (
	OpLogin       = "login"
	OpLogout      = "logout"
	OpCheckLogin  = "checklogin"
	OpCheckLogout = "checklogout"
	OpGetList     = "getList"
)

var Broadcaster = &broadcast{
	users:          make(map[string]*User),
	ops:            make(chan broadcastOp),
	messageChannel: make(chan *Message, setting.MessageQueueLength),
}

func (b *broadcast) Start() {
	for {
		select {
		case op := <-b.ops:
			switch op.typ {
			case OpLogin:
				b.users[op.user.Name] = op.user
				UserMessageProcessor.Send(op.user)
				b.Broadcast(NewLoginMsg(op.user))

			case OpLogout:
				delete(b.users, op.user.Name)
				op.user.CloseChannel()
				op.user.IsOnline = false
				b.Broadcast(NewLogoutMsg(op.user))

			case OpCheckLogin:
				_, exists := b.users[op.user.Name]
				op.reply <- !exists

			case OpCheckLogout:
				_, exists := b.users[op.user.Name]
				op.reply <- exists

			case OpGetList:
				usersList := make([]*User, 0, len(b.users))
				for _, user := range b.users {
					usersList = append(usersList, user)
				}
				op.reply <- usersList
			}

		case msg := <-b.messageChannel:
			for _, user := range b.users {
				// log.Println(user.Name)
				if user.ID == msg.User.ID && msg.Type != MsgTypeNormal {
					continue
				}
				// log.Println("sending msg to user channel!")
				// log.Printf("msg to channel:%v", msg)
				user.MessageChannel <- msg
			}
			UserMessageProcessor.Save(msg)
		}
	}
}

func (b *broadcast) UserLogin(user *User) {
	b.ops <- broadcastOp{typ: OpLogin, user: user}
}

func (b *broadcast) UserLogout(user *User) {
	b.ops <- broadcastOp{typ: OpLogout, user: user}
}

// we use channel to ensure concurrent safety
func (b *broadcast) CheckUserCanLogin(name string) bool {
	reply := make(chan interface{})
	b.ops <- broadcastOp{typ: OpCheckLogin, user: &User{Name: name}, reply: reply}
	boolReply, _ := (<-reply).(bool)
	return boolReply
}

func (b *broadcast) CheckUserCanLogout(name string) bool {
	reply := make(chan interface{})
	b.ops <- broadcastOp{typ: OpCheckLogout, user: &User{Name: name}, reply: reply}
	boolReply, _ := (<-reply).(bool)
	return boolReply
}

func (b *broadcast) GetUserList() []*User {
	reply := make(chan interface{})
	b.ops <- broadcastOp{typ: OpGetList, reply: reply}
	usersReply, _ := (<-reply).([]*User)
	return usersReply
}

func (b *broadcast) Broadcast(msg *Message) {
	if len(b.messageChannel) >= setting.MessageQueueLength {
		log.Println("the broadcast queue has been full")
	} else {
		// log.Println("broadcast successfully!")
		b.messageChannel <- msg
	}
}

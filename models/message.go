package models

import (
	"fmt"
	"strings"
	"time"
)

const (
	MsgTypeNormal = iota
	MsgTypeWelcome
	MsgTypeUserLogin
	MsgTypeUserLogout
	MsgTypeError
	MsgTypeUserList
)

type Message struct {
	User    *User  `json:"from_user"`
	Type    int    `json:"type"`
	Content string `json:"content"`

	CreatedAt time.Time `json:"created_at"`
	Ats       []string  `json:"ats"`
}

func NewMessage(user *User, msgType int, content string) *Message {
	msg := &Message{
		User:      user,
		Type:      msgType,
		Content:   content,
		CreatedAt: time.Now(),
	}

	return msg
}

func NewWelcomeMsg(user *User) *Message {
	return NewMessage(user,
		MsgTypeWelcome,
		fmt.Sprintf("hello: %s ,welcome to the chatroom!", user.Name))
}

func NewLoginMsg(user *User) *Message {
	return NewMessage(user,
		MsgTypeUserLogin,
		fmt.Sprintf("%s has entered the chatroom!", user.Name))
}

func NewLogoutMsg(user *User) *Message {
	return NewMessage(user,
		MsgTypeUserLogout,
		fmt.Sprintf("%s has exited the chatroom!", user.Name))
}

func NewErrorMsg(content string) *Message {
	return NewMessage(System,
		MsgTypeError,
		content)
}

func NewUserListMessage(users []*User) *Message {
	userNames := make([]string, 0, len(users))
	for _, user := range users {
		userNames = append(userNames, user.Name)
	}

	content := "Current users: " + strings.Join(userNames, ", ")

	return NewMessage(System,
		MsgTypeUserList,
		content)
}

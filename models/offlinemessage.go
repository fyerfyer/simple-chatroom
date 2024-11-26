package models

import (
	"container/list"

	"github.com/fyerfyer/chatroom/pkg/setting"
)

type userMessageProcessor struct {
	maxMsgNum int

	// the front of the deque stores the oldest message
	recentMsgDeque *list.List
	userMsgDeque   map[string]*list.List
}

var UserMessageProcessor = newUserMessageProcessor()

func newUserMessageProcessor() *userMessageProcessor {
	return &userMessageProcessor{
		maxMsgNum:      setting.OfflineMsgNum,
		recentMsgDeque: list.New(),
		userMsgDeque:   make(map[string]*list.List),
	}
}

func (p *userMessageProcessor) Save(msg *Message) {
	if msg.Type != MsgTypeNormal {
		return
	}

	if p.recentMsgDeque.Len() >= p.maxMsgNum {
		p.recentMsgDeque.Remove(p.recentMsgDeque.Front())
	}
	p.recentMsgDeque.PushBack(msg)

	// deal with the '@' operations
	for _, name := range msg.Ats {
		name = name[1:]
		var (
			userMsg *list.List
			ok      bool
		)

		if userMsg, ok = p.userMsgDeque[name]; !ok {
			userMsg = list.New()
		}
		userMsg.PushBack(msg)
		p.userMsgDeque[name] = userMsg
	}
}

func (p *userMessageProcessor) Send(user *User) {
	// send the recent message to the user
	for msg := p.recentMsgDeque.Front(); msg != nil; msg = msg.Next() {
		if msg.Value != nil {
			user.MessageChannel <- msg.Value.(*Message)
		}
	}

	// if user is offline
	// there's no need to send the @ message to it
	if user.IsNew {
		return
	}

	userMsg, ok := p.userMsgDeque[user.Name]
	if ok {
		for msg := userMsg.Front(); msg != nil; msg = msg.Next() {
			user.MessageChannel <- msg.Value.(*Message)
		}

		delete(p.userMsgDeque, user.Name)
	}
}

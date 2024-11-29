package models

import (
	"container/list"
	"strconv"
	"testing"
)

func ClearUserMsgProcessorForTesting() {
	UserMessageProcessor.recentMsgDeque = list.New()
	UserMessageProcessor.userMsgDeque = make(map[string]*list.List)
}

func TestOfflineSave(t *testing.T) {
	defer ClearUserMsgProcessorForTesting()

	var msgs []*Message
	wantedMsg := "testing message:"
	user := &User{Name: "testing_user1"}
	atUser := &User{Name: "testing_user2"}

	for i := 0; i < 11; i++ {
		msg := NewMessage(user, MsgTypeNormal, wantedMsg+strconv.Itoa(i))
		msg.Ats = append(msg.Ats, "@"+atUser.Name)
		msgs = append(msgs, msg)
	}

	// test normal save
	UserMessageProcessor.Save(msgs[0])
	msg, _ := UserMessageProcessor.recentMsgDeque.Front().Value.(*Message)
	if msg.Content != msgs[0].Content {
		t.Errorf("wanted message %v in recentMsgQueue, but got %v", msg.Content, msgs[0].Content)
		return
	}
	if UserMessageProcessor.userMsgDeque[atUser.Name].Len() == 0 {
		t.Error("@user's message should be saved in UserMsgQueue")
		return
	}

	// test message pop
	for i := 1; i < 11; i++ {
		UserMessageProcessor.Save(msgs[i])
	}

	msg, _ = UserMessageProcessor.recentMsgDeque.Front().Value.(*Message)
	if msg.Content == msgs[0].Content {
		t.Error("the first message should be popped")
		return
	}

	if UserMessageProcessor.recentMsgDeque.Len() != 10 {
		t.Error("recent msg queue should contain 10 messages")
		return
	}
}

func TestOfflineSend(t *testing.T) {
	// create an online user & an offline user
	userOnline := &User{
		Name:           "testing_user_online",
		MessageChannel: make(chan *Message, 32),
		IsOnline:       true,
	}

	userOffline := &User{
		Name:           "testing_user_offline",
		MessageChannel: make(chan *Message, 32),
		IsOnline:       false,
	}

	wantedMsg := "testing message:"
	for i := 0; i < 3; i++ {
		msg := NewMessage(userOnline, MsgTypeNormal, wantedMsg+strconv.Itoa(i))
		msg.Ats = append(msg.Ats, "@"+userOnline.Name)
		msg.Ats = append(msg.Ats, "@"+userOffline.Name)
		UserMessageProcessor.Save(msg)
	}

	// t.Log(models.GetUserMsgQueueForTesting()[userOnline.Name].Len())

	UserMessageProcessor.Send(userOnline)
	UserMessageProcessor.Send(userOffline)

	if len(UserMessageProcessor.userMsgDeque) > 1 {
		t.Error("after sending, UserMsgQueue for online user should be deleted")
		return
	}

	if UserMessageProcessor.userMsgDeque[userOffline.Name].Len() != 3 {
		t.Error("after sending, messages should be in offline user's channel")
		return
	}
}

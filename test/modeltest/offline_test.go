package modeltest

import (
	"strconv"
	"testing"

	"github.com/fyerfyer/chatroom/models"
)

func TestOfflineSave(t *testing.T) {
	defer models.ClearUserMsgProcessorForTesting()

	var msgs []*models.Message
	wantedMsg := "testing message:"
	user := &models.User{Name: "testing_user1"}
	atUser := &models.User{Name: "testing_user2"}

	for i := 0; i < 11; i++ {
		msg := models.NewMessage(user, models.MsgTypeNormal, wantedMsg+strconv.Itoa(i))
		msg.Ats = append(msg.Ats, "@"+atUser.Name)
		msgs = append(msgs, msg)
	}

	// test normal save
	models.TestUserMessageProcessor.Save(msgs[0])
	msg, _ := models.GetRecentMsgQueueForTesting().Front().Value.(*models.Message)
	if msg.Content != msgs[0].Content {
		t.Errorf("wanted message %v in recentMsgQueue, but got %v", msg.Content, msgs[0].Content)
		return
	}
	if models.GetUserMsgQueueForTesting()[atUser.Name].Len() == 0 {
		t.Error("@user's message should be saved in UserMsgQueue")
		return
	}

	// test message pop
	for i := 1; i < 11; i++ {
		models.TestUserMessageProcessor.Save(msgs[i])
	}

	msg, _ = models.GetRecentMsgQueueForTesting().Front().Value.(*models.Message)
	if msg.Content == msgs[0].Content {
		t.Error("the first message should be popped")
		return
	}

	if models.GetRecentMsgQueueForTesting().Len() != 10 {
		t.Error("recent msg queue should contain 10 messages")
		return
	}
}

func TestOfflineSend(t *testing.T) {
	// create an online user & an offline user
	userOnline := &models.User{
		Name:           "testing_user_online",
		MessageChannel: make(chan *models.Message, 32),
		IsOnline:       true,
	}

	userOffline := &models.User{
		Name:           "testing_user_offline",
		MessageChannel: make(chan *models.Message, 32),
		IsOnline:       false,
	}

	wantedMsg := "testing message:"
	for i := 0; i < 3; i++ {
		msg := models.NewMessage(userOnline, models.MsgTypeNormal, wantedMsg+strconv.Itoa(i))
		msg.Ats = append(msg.Ats, "@"+userOnline.Name)
		msg.Ats = append(msg.Ats, "@"+userOffline.Name)
		models.TestUserMessageProcessor.Save(msg)
	}

	// t.Log(models.GetUserMsgQueueForTesting()[userOnline.Name].Len())

	models.TestUserMessageProcessor.Send(userOnline)
	models.TestUserMessageProcessor.Send(userOffline)

	if len(models.GetUserMsgQueueForTesting()) > 1 {
		t.Error("after sending, UserMsgQueue for online user should be deleted")
		return
	}

	if models.GetUserMsgQueueForTesting()[userOffline.Name].Len() != 3 {
		t.Error("after sending, messages should be in offline user's channel")
		return
	}
}

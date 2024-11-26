package setting

import (
	"log"

	"github.com/fyerfyer/chatroom/pkg/utils"
	"github.com/go-ini/ini"
)

var (
	Cfg *ini.File

	HTTPPort               string
	MessageQueueLength     int
	OfflineMsgNum          int
	UserMessageQueueLength int
)

func init() {
	var err error
	filepath := utils.InferRootDir() + "/conf/chatroom.ini"
	Cfg, err = ini.Load(filepath)
	if err != nil {
		log.Fatalf("Failed to parse %v: %v", filepath, err)
	}

	var chatroom = Cfg.Section("chatroom")
	var server = Cfg.Section("server")
	HTTPPort = server.
		Key("HTTP_PORT").String()

	MessageQueueLength = chatroom.
		Key("Message_Queue_Length").
		MustInt(1024)

	OfflineMsgNum = chatroom.
		Key("Offline_Message_Num").
		MustInt(10)

	UserMessageQueueLength = chatroom.
		Key("UserMessageQueueLength").
		MustInt(32)
	// log.Println(HTTPPort)
	// log.Println(MessageQueueLength)
	// log.Println(OfflineMsgNum)
}

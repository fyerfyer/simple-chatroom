package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var serverURL = "ws://localhost:8000/ws?"
var clientname string

func main() {
	// Command line flag for client name
	flag.StringVar(&clientname, "name", "Alice", "the chatroom login name")
	flag.Parse()

	// Connect to WebSocket server
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, serverURL+"name="+clientname, nil)
	if err != nil {
		log.Fatal("Failed to set up client:", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "client exit")

	// Message reader
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter message (or 'exit' to quit): ")
		input, _ := reader.ReadString('\n')
		input = input[:len(input)-1]

		if input == "exit" {
			sendMessage(conn, "exit", false) // Exit message
			break
		}

		sendMessage(conn, input, true) // Regular message
	}
}

func sendMessage(conn *websocket.Conn, content string, isNew bool) {
	msg := map[string]interface{}{
		"id":         1,
		"name":       clientname,
		"created_at": time.Now(),
		"IsNew":      isNew,
		"content":    content,
	}

	// Send the message
	msgCtx, msgCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer msgCancel()

	if err := wsjson.Write(msgCtx, conn, msg); err != nil {
		log.Println("Failed to send message:", err)
		return
	}

	// Receive the response
	var reply map[string]interface{}
	if err := wsjson.Read(msgCtx, conn, &reply); err != nil {
		log.Println("Failed to receive message:", err)
	} else {
		log.Println("Received message:", reply)
	}
}

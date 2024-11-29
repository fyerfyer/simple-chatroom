package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fyerfyer/chatroom/pkg/setting"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var clientname string
var serverURL = "ws://localhost" + fmt.Sprintf(":"+setting.HTTPPort) + "/ws?name="

func main() {
	flag.StringVar(&clientname, "name", "Alice", "the chatroom login name")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, serverURL+clientname, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "client closed")

	fmt.Println("Connected to the server. Type messages to send")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Receiver goroutine exiting...")
				return
			default:
				var msg interface{}
				err := wsjson.Read(ctx, conn, &msg)
				if err != nil {
					log.Printf("Error reading message: %v", err)
					cancel()
					return
				}
				fmt.Printf("Received: %v\n", msg)
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "exit" {
			fmt.Println("Exiting...")
			cancel()
			break
		}

		err := wsjson.Write(ctx, conn, text)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			cancel()
			break
		}
	}

	time.Sleep(1 * time.Second)
	log.Println("Client stopped.")
}

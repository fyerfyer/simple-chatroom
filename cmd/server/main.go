package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fyerfyer/chatroom/pkg/setting"
	"github.com/fyerfyer/chatroom/routers"

	_ "net/http/pprof"
)

func main() {
	fmt.Printf("Welcome to ChatRoom!!!\n")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":" + setting.HTTPPort),
		Handler: routers.InitRouter(),
	}

	log.Printf("serving on port: %v", setting.HTTPPort)

	log.Fatal(srv.ListenAndServe())
}

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/vpatsenko/pow-server/internal/pkg/cache"
	"github.com/vpatsenko/pow-server/internal/pkg/config"
	"github.com/vpatsenko/pow-server/internal/server"
)

func main() {
	fmt.Println("start server")
	cfg, err := config.Load("config/config.json")
	if err != nil {
		fmt.Println("error load config:", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())

	c := cache.NewCache()

	rand.Seed(time.Now().UnixNano())
	log.Printf("starting server with the following config: \n %+v", cfg)

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt)

	srv := server.NewServer(cfg, c)
	go func() {
		<-shutdownChan
		cancel()
	}()

	err = srv.Run(ctx)
	if err != nil {
		fmt.Println("server error:", err)
	}

}

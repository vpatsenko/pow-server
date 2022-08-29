package main

import (
	"fmt"

	"github.com/vpatsenko/pow-server/internal/client"
	"github.com/vpatsenko/pow-server/internal/pkg/config"
)

func main() {
	fmt.Println("start client")

	cfg, err := config.Load("config/config.json")
	if err != nil {
		fmt.Println("error load config:", err)
		return
	}

	err = client.Run(cfg)
	if err != nil {
		fmt.Println("client error:", err)
	}
}

package main

import (
	"os"
	"sock5"
)

func main() {
	defaultPort := os.Getenv("DEFAULT_PORT")
	if len(defaultPort) <= 0 {
		defaultPort = "10008"
	}

	sock5Server, err := sock5.NewSocks5Server(":" + defaultPort)
	if err != nil {
		panic(err)
	}

	sock5Server.Start()
}

package main

import (
	"os"
	"sock5-go"
)

func main() {
	defaultPort := os.Getenv("DEFAULT_PORT")
	if len(defaultPort) <= 0 {
		defaultPort = "10008"
	}

	sock5Server, err := sock5.NewSocks5ServerConfigurable(":"+defaultPort, sock5.ServerConfiguration{
		Mode: "auth",
	})
	if err != nil {
		panic(err)
	}

	sock5Server.Start()
}

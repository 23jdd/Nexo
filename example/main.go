package main

import "github.com/23jdd/Nexo/server"

func main() {
	httpServer := server.New()
	if err := httpServer.Run("127.0.0.1", 9999); err != nil {
		panic(err)
	}
}

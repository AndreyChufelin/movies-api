package main

import (
	"time"

	"github.com/AndreyChufelin/movies-api/internal/server/rest"
)

func main() {
	restServer := rest.NewServer("localhost", "1323", time.Minute, 10*time.Second, 30*time.Second)
	restServer.Start()
}

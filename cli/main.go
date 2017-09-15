package main

import (
	"log"
	"time"

	"github.com/djavorszky/disco"
)

var (
	target = "224.0.0.1:9999"
)

func main() {
	c, err := disco.Subscribe(target)
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	err = disco.Broadcast(target, "Yo mamma")
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	log.Printf("%#v", <-c)
}

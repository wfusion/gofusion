package main

import (
	"log"
	"time"

	"github.com/wfusion/gofusion/common/infra/watermill/message"
)

func main() {
	msg := message.NewMessage("1", []byte("foo"))

	go func() {
		time.Sleep(time.Millisecond * 10)
		msg.Ack()
	}()

	select {
	case <-msg.Acked():
		log.Print("ack received")
	case <-msg.Nacked():
		log.Print("nack received")
	}
}

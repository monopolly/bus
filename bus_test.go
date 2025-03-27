package bus

//testing
import (
	"fmt"
	"log"
	"strings"
	"testing"
	"time"
)

// docker run -p 4222:4222 -ti nats:latest -js

var (
	token     = ""
	host      = "127.0.0.1"
	storename = "test"
	c, _      = New(host, token, storename)
)

func TestStore(u *testing.T) {
	__(u)

	err := c.AddInt("int", 42)
	if err != nil {
		log.Println(err)
		panic("add int")
	}

	i, err := c.Int("int")
	if err != nil {
		log.Println(err)
		panic("int")
	}

	if i != 42 {
		panic("int")
	}

}

func TestQueue(u *testing.T) {
	__(u)

	name := "testmail2"

	Wait(c)

	log.Println("create queue...")
	q, err := c.Queue(name, name+".*")
	if err != nil {
		log.Println("create queue:", err)
		return
	}

	err = q.Group(name+".send", func(topic string, body []byte) (done bool) {
		log.Println("send1:", topic, string(body))
		return true
	})
	if err != nil {
		panic(err)
	}

	q.Group(name+".send", func(topic string, body []byte) (done bool) {
		log.Println("send2:", topic, string(body))
		return true
	})
	q.Group(name+".send", func(topic string, body []byte) (done bool) {
		log.Println("send3:", topic, string(body))
		return true
	})
	q.Group(name+".confirm", func(topic string, body []byte) (done bool) {
		log.Println("confirm1:", topic, string(body))
		return true
	})
	q.Group(name+".confirm", func(topic string, body []byte) (done bool) {
		log.Println("confirm2:", topic, string(body))
		return true
	})

	var count int
	go func() {
		c2, _ := New(host, token, storename)
		for range 10 {
			time.Sleep(time.Second)
			count++
			c2.Publish(name+".send", fmt.Appendf([]byte{}, "%d", count))
			count++
			c2.Publish(name+".confirm", fmt.Appendf([]byte{}, "%d", count))
		}
	}()

	select {}
}

func __(u *testing.T) {
	fmt.Printf("\033[1;32m%s\033[0m\n", strings.ReplaceAll(u.Name(), "Test", ""))
}

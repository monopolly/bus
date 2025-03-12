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
)

func TestEngine(u *testing.T) {
	__(u)

	p, err := New(host, token, storename)
	if err != nil {
		panic(err)
	}

	err = p.AddInt("int", 42)
	if err != nil {
		log.Println(err)
		panic("add int")
	}

	i, err := p.Int("int")
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

	name := "qqq"
	// log.Println("connect...")
	c, err := New(host, token, storename)
	if err != nil {
		log.Println("queue:", err)
		return
	}

	log.Println("create queue...")
	p, err := c.Queue(name, name+".*")
	if err != nil {
		log.Println("create queue:", err)
		return
	}

	log.Println("s1")
	p.Subscribe("ios", name+".ios", func(topic string, body []byte) (done bool) {
		log.Println("s1ios:", topic, string(body))
		time.Sleep(time.Second * 7)
		return true
	})

	log.Println("publish first...")
	p.Publish(name+".ios", []byte("ios"))
	time.Sleep(time.Second)
	p.Publish(name+".android", []byte("android"))
	time.Sleep(time.Second)
	p.Publish(name+".win", []byte("win"))
	time.Sleep(time.Second)

	// log.Println("connect...")
	c2, err := New(host, token, storename)
	if err != nil {
		log.Println("queue:", err)
		return
	}
	p2, err := c2.Queue(name, name+".*")
	if err != nil {
		log.Println("create queue:", err)
		return
	}

	p2.Subscribe("ios", name+".ios", func(topic string, body []byte) (done bool) {
		log.Println("g1ios:", topic, string(body))
		time.Sleep(time.Second * 2)
		return true
	})
	p.Subscribe("android", name+".android", func(topic string, body []byte) (done bool) {
		log.Println("s3android", topic, string(body))
		return true
	})
	p.Subscribe("win", name+".win", func(topic string, body []byte) (done bool) {
		log.Println("s3win", topic, string(body))
		return true
	})

	time.Sleep(time.Second)

	go func() {
		log.Println("publishing...")
		for {
			time.Sleep(time.Second)
			log.Println("queue")
			p.Publish(name+".ios", []byte(time.Now().String()))

		}
	}()

	select {}
}

func __(u *testing.T) {
	fmt.Printf("\033[1;32m%s\033[0m\n", strings.ReplaceAll(u.Name(), "Test", ""))
}

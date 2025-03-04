package bus

//testing
import (
	"fmt"
	"log"
	"strings"
	"testing"
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

	select {}
}

func __(u *testing.T) {
	fmt.Printf("\033[1;32m%s\033[0m\n", strings.ReplaceAll(u.Name(), "Test", ""))
}

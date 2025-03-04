package bus

import (
	"context"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Message = *nats.Msg

type Stream = jetstream.JetStream

func New(token, host, storename string) (a *Engine, err error) {
	a = new(Engine)
	a.token = token
	a.host = host
	a.storename = storename
	err = a.init()
	return
}

type Engine struct {
	conn      *nats.Conn
	token     string
	host      string
	storename string

	stream Stream
	store  *Store
}

func (a *Engine) init() (err error) {

	// connection
	for {
		c, err := nats.Connect(a.host, nats.Token(a.token))
		if err != nil {
			time.Sleep(time.Second * 5)
			log.Println("bus error:", err)
			continue
		}

		a.conn = c
		break
	}

	// init stream
	a.stream, err = jetstream.New(a.conn)
	if err != nil {
		return
	}
	// default store
	a.store = a.GetStore(a.storename)
	return
}

// time.*.east or time.us.>
func (a *Engine) Publish(to string, res []byte) error {
	return a.conn.Publish(to, res)
}

// time.*.east or time.us.>
func (a *Engine) Request(to string, res []byte) (resp []byte, err error) {
	m, err := a.conn.Request(to, res, time.Second*30)
	if err != nil {
		return
	}
	resp = m.Data
	return
}

// time.*.east or time.us.>
func (a *Engine) Subscribe(to string, v func([]byte)) (err error) {
	_, err = a.conn.Subscribe(to, func(msg *nats.Msg) {
		v(msg.Data)
	})
	return
}

// time.*.east or time.us.>
func (a *Engine) SubscribeMessage(to string, v func(Message)) (err error) {
	_, err = a.conn.Subscribe(to, func(msg *nats.Msg) {
		v(msg)
	})
	return
}

// time.*.east or time.us.>
func (a *Engine) Group(to, group string, v func([]byte)) (err error) {
	_, err = a.conn.QueueSubscribe(to, group, func(msg *nats.Msg) {
		v(msg.Data)
	})
	return
}

// time.*.east or time.us.>
func (a *Engine) GroupMessage(to, group string, v func(Message)) (err error) {
	_, err = a.conn.QueueSubscribe(to, group, func(msg *nats.Msg) {
		v(msg)
	})
	return
}

// store
func (a *Engine) Add(k string, v []byte) (err error) {
	return a.store.Add(k, v)
}

// store
func (a *Engine) AddString(k, v string) (err error) {
	return a.store.AddString(k, v)
}

// store
func (a *Engine) AddInt(k string, v int) (err error) {
	return a.store.AddInt(k, v)
}

// store
func (a *Engine) Get(k string) (res []byte, err error) {
	return a.store.Get(k)
}

// store
func (a *Engine) String(k string) (res string, err error) {
	return a.store.String(k)
}

// store
func (a *Engine) Int(k string) (res int, err error) {
	return a.store.Int(k)
}

func (a *Engine) GetStore(name string) (res *Store) {

	// stream
	if a.stream == nil {
		return
	}

	var store Store
	store.store, _ = a.stream.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{Bucket: name})
	res = &store
	return

}

func (a *Engine) CreateTTLStore(name string, ttl time.Duration) (res *Store) {

	// stream
	if a.stream == nil {
		return
	}

	var store Store
	store.store, _ = a.stream.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{Bucket: name, TTL: ttl})
	res = &store
	return

}

func (a *Engine) CreateMemoryStore(name string, ttl ...time.Duration) (res *Store, err error) {

	// stream
	if a.stream == nil {
		return
	}

	var store Store

	switch ttl != nil {
	case true:
		store.store, err = a.stream.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{Bucket: name, Storage: jetstream.MemoryStorage, TTL: ttl[0]})
	default:
		store.store, err = a.stream.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{Bucket: name, Storage: jetstream.MemoryStorage})
	}

	if err != nil {
		return
	}

	res = &store
	return

}

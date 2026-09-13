package bus

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// docker run -p 4222:4222 -ti nats:latest -js
// BUS_HOST=127.0.0.1:4222 go test ./...

func connect(t *testing.T) *Engine {
	t.Helper()
	host := os.Getenv("BUS_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	c, err := New(host, os.Getenv("BUS_TOKEN"), suffix("teststore"))
	if err != nil {
		t.Skipf("nats server is not available at %s: %v", host, err)
	}
	t.Cleanup(func() {
		c.JetStream().DeleteKeyValue(context.Background(), c.storename)
		c.Close()
	})
	return c
}

func suffix(s string) string {
	return fmt.Sprintf("%s%d", s, time.Now().UnixNano()%1_000_000_000)
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout: %s", msg)
}

func TestNoConnection(t *testing.T) {
	c, err := New("127.0.0.1:1", "", "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if c != nil {
		t.Fatal("engine must be nil on error")
	}

	// nil engine must not panic and must not hang
	Wait(c)
	c.Close()
	if err := c.Publish("a", nil); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
	if _, err := c.Queue("q", "q.*"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
	if err := c.Add("k", nil); !errors.Is(err, ErrNoStore) {
		t.Fatalf("expected ErrNoStore, got %v", err)
	}

	var s Stream
	if err := s.Publish("a", nil); !errors.Is(err, ErrNoStream) {
		t.Fatalf("expected ErrNoStream, got %v", err)
	}
}

func TestUnique(t *testing.T) {
	cases := map[string]string{
		"signup.send": "signupsend",
		"signup.*":    "signup_any_",
		"signup.>":    "signup_all_",
		"a-b_c":       "abc",
	}
	for in, want := range cases {
		if got := unique(in); got != want {
			t.Errorf("unique(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBytesInt(t *testing.T) {
	if BytesInt(IntBytes(-42)) != -42 {
		t.Fatal("roundtrip")
	}
	if BytesInt(nil) != 0 || BytesInt([]byte("5")) != 0 {
		t.Fatal("short input must give 0")
	}
}

func TestStore(t *testing.T) {
	c := connect(t)

	if err := c.AddInt("int", 42); err != nil {
		t.Fatal(err)
	}
	i, err := c.Int("int")
	if err != nil || i != 42 {
		t.Fatalf("int: %d %v", i, err)
	}

	if err := c.AddString("str", "5"); err != nil {
		t.Fatal(err)
	}
	s, err := c.String("str")
	if err != nil || s != "5" {
		t.Fatalf("string: %q %v", s, err)
	}
	if _, err := c.Int("str"); !errors.Is(err, ErrInvalidInt) {
		t.Fatalf("expected ErrInvalidInt, got %v", err)
	}

	keys, err := c.DefaultStore().Keys()
	if err != nil || len(keys) != 2 {
		t.Fatalf("keys: %v %v", keys, err)
	}

	if err := c.DefaultStore().Delete("int"); err != nil {
		t.Fatal(err)
	}
	if err := c.DefaultStore().Delete("str"); err != nil {
		t.Fatal(err)
	}
	if c.DefaultStore().Exists("int") {
		t.Fatal("deleted key exists")
	}
	keys, err = c.DefaultStore().Keys()
	if err != nil || keys == nil || len(keys) != 0 {
		t.Fatalf("empty keys: %v %v", keys, err)
	}
}

func TestStoreKinds(t *testing.T) {
	c := connect(t)

	m, err := c.CreateMemoryStore(suffix("mem"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.JetStream().DeleteKeyValue(context.Background(), m.KeyValue().Bucket())
	if err := m.AddString("a", "b"); err != nil {
		t.Fatal(err)
	}

	ttl, err := c.CreateTTLStore(suffix("ttl"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer c.JetStream().DeleteKeyValue(context.Background(), ttl.KeyValue().Bucket())
	if err := ttl.AddString("a", "b"); err != nil {
		t.Fatal(err)
	}
}

func TestWatch(t *testing.T) {
	c := connect(t)
	store := c.DefaultStore()

	// existing value must not be reported, only updates
	if err := store.AddString("mail.host", "old"); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	got := map[string][]string{}
	record := func(name string) func(k string, v []byte) {
		return func(k string, v []byte) {
			mu.Lock()
			defer mu.Unlock()
			got[name] = append(got[name], k+"="+string(v))
		}
	}

	if err := store.Watch("mail.*", record("watch")); err != nil {
		t.Fatal(err)
	}
	if err := store.Updates(record("all")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)

	store.AddString("mail.host", "new")
	store.AddString("mail.port", "25")
	store.AddString("other", "x")
	store.Delete("mail.port")

	waitFor(t, 5*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got["watch"]) == 3 && len(got["all"]) == 4
	}, "watch and updates events delivered")

	mu.Lock()
	defer mu.Unlock()
	want := []string{"mail.host=new", "mail.port=25", "mail.port="}
	for i, w := range want {
		if got["watch"][i] != w {
			t.Fatalf("watch[%d] = %q, want %q (all: %v)", i, got["watch"][i], w, got["watch"])
		}
	}
}

func TestQueue(t *testing.T) {
	c := connect(t)

	name := suffix("testmail")
	q, err := c.Queue(name, name+".*")
	if err != nil {
		t.Fatal(err)
	}
	defer c.JetStream().DeleteStream(context.Background(), name)

	var mu sync.Mutex
	seen := map[string]int{}
	var workers [3]atomic.Int32
	handler := func(i int) func(topic string, body []byte) bool {
		return func(topic string, body []byte) bool {
			workers[i].Add(1)
			mu.Lock()
			seen[topic+":"+string(body)]++
			mu.Unlock()
			return true
		}
	}

	for i := range 3 {
		if err := q.Group(name+".send", handler(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.Group(name+".confirm", handler(0)); err != nil {
		t.Fatal(err)
	}

	const n = 20
	for i := range n {
		if err := q.Publish(name+".send", fmt.Appendf(nil, "%d", i)); err != nil {
			t.Fatal(err)
		}
		if err := q.Publish(name+".confirm", fmt.Appendf(nil, "%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	waitFor(t, 10*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(seen) == 2*n
	}, "all queue messages delivered")

	// exactly once
	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	for k, v := range seen {
		if v != 1 {
			t.Errorf("%s delivered %d times", k, v)
		}
	}
	mu.Unlock()

	// publish to a subject without a stream must fail instead of silently dropping
	if err := q.Publish("nostream."+name, nil); err == nil {
		t.Fatal("expected error for a subject without a stream")
	}
}

func TestQueueRedelivery(t *testing.T) {
	c := connect(t)
	old := NakDelay
	NakDelay = 100 * time.Millisecond
	defer func() { NakDelay = old }()

	name := suffix("retry")
	q, err := c.Queue(name, name+".*")
	if err != nil {
		t.Fatal(err)
	}
	defer c.JetStream().DeleteStream(context.Background(), name)

	var attempts atomic.Int32
	err = q.GroupDoubleAck(name+".job", func(topic string, body []byte) bool {
		return attempts.Add(1) >= 3
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Publish(name+".job", []byte("x")); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 5*time.Second, func() bool { return attempts.Load() == 3 }, "message redelivered until done")
	time.Sleep(300 * time.Millisecond)
	if attempts.Load() != 3 {
		t.Fatalf("delivered %d times after ack", attempts.Load())
	}
}

func TestStream(t *testing.T) {
	c := connect(t)

	if _, err := c.Stream("nosubj"); !errors.Is(err, ErrNoSubjects) {
		t.Fatalf("expected ErrNoSubjects, got %v", err)
	}

	name := suffix("events")
	s, err := c.Stream(name, name+".>")
	if err != nil {
		t.Fatal(err)
	}
	defer c.JetStream().DeleteStream(context.Background(), name)

	var a, b atomic.Int32
	if err := s.Subscribe(name+".user.*", func(topic string, body []byte) bool { a.Add(1); return true }); err != nil {
		t.Fatal(err)
	}
	if err := s.Subscribe(name+".user.*", func(topic string, body []byte) bool { b.Add(1); return true }); err != nil {
		t.Fatal(err)
	}

	for i := range 5 {
		if err := s.Publish(name+".user.signup", fmt.Appendf(nil, "%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	// every subscriber gets every message
	waitFor(t, 5*time.Second, func() bool { return a.Load() == 5 && b.Load() == 5 }, "fan-out delivery")
}

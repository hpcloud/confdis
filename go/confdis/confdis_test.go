package confdis

import (
	"github.com/vmihailenco/redis"
	"net"
	"testing"
	"time"
)

type SampleConfig struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
	Meta  struct {
		Researcher string `json:"researcher"`
		Grant      int    `json:"grant"`
	} `json:"meta"`
}

func NewConfDis(t *testing.T, rootKey string) *ConfDis {
	redis, err := NewRedisClient("localhost:6379", "", 0)
	if err != nil {
		t.Fatalf("Unable to connect to redis: %v", err)
	}

	c, err := New(redis, "test:confdis:simple", SampleConfig{})
	if err != nil {
		t.Fatalf("Failed to connect to redis: %v", err)
	}
	return c
}

func redisDelay() {
	// Allow reasonable delay for network/redis latency in the above
	// save operation.
	time.Sleep(time.Duration(100 * time.Millisecond))
}

func TestSimple(t *testing.T) {
	c := NewConfDis(t, "test:confdis:simple")
	if err := c.AtomicSave(func(i interface{}) error {
		config := i.(*SampleConfig)
		config.Name = "primates"
		config.Users = []string{"chimp", "bonobo", "lemur"}
		config.Meta.Researcher = "Jane Goodall"
		config.Meta.Grant = 1200
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestChangeNotification(t *testing.T) {
	// First client, with initial data.
	c := NewConfDis(t, "test:confdis:notify")
	go c.MustReceiveChanges()
	if err := c.AtomicSave(func(i interface{}) error {
		config := i.(*SampleConfig)
		config.Name = "primates-changes"
		config.Users = []string{"chimp", "bonobo", "lemur"}
		config.Meta.Researcher = "Jane Goodall"
		config.Meta.Grant = 1200
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	redisDelay()

	// Second client
	c2 := NewConfDis(t, "test:confdis:notify")
	go c2.MustReceiveChanges()

	if c2.Config.(*SampleConfig).Meta.Researcher != "Jane Goodall" {
		t.Fatal("different value")
	}

	// Trigger a change via the first client
	if err := c.AtomicSave(func(i interface{}) error {
		config := i.(*SampleConfig)
		config.Meta.Researcher = "Francine Patterson"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	redisDelay()

	// Second client must get notified of that change
	if c2.Config.(*SampleConfig).Meta.Researcher != "Francine Patterson" {
		t.Fatal("did not receive change")
	}
}

func TestAtomicSave(t *testing.T) {
	// First client, with initial data.
	c := NewConfDis(t, "test:confdis:atomicsave")
	go c.MustReceiveChanges()
	if err := c.AtomicSave(func(i interface{}) error {
		config := i.(*SampleConfig)
		config.Name = "primates-changes"
		config.Users = []string{"chimp", "bonobo", "lemur"}
		config.Meta.Researcher = "Jane Goodall"
		config.Meta.Grant = 1200
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	redisDelay()

	// Trigger a change every 20 milliseconds
	go func() {
		for _ = range time.Tick(20 * time.Millisecond) {
			if err := c.AtomicSave(func(i interface{}) error {
				config := i.(*SampleConfig)
				config.Meta.Grant += 15
				return nil
			}); err != nil {
				t.Fatalf("Error in periodic-saving: %v", err)
			}
		}
	}()

	// Second client
	c2 := NewConfDis(t, "test:confdis:atomicsave")
	go c2.MustReceiveChanges()

	// Trigger a *slow* change, expecting write conflict.
	if err := c2.AtomicSave(func(i interface{}) error {
		config := i.(*SampleConfig)
		// Choose a delay value (50ms) greater than the frequency of
		// change (20ms) from the other client above.
		time.Sleep(50 * time.Millisecond)
		config.Meta.Researcher = "Francine Patterson"
		return nil
	}); err == nil {
		t.Fatal("Expecting this save to fail.")
	} else {
		// t.Logf("Failed as expected with: %v", err)
	}
}

// NewRedisClient connects to redis after ensuring that the server is
// indeed running.
func NewRedisClient(addr, password string, database int64) (*redis.Client, error) {
	// Bug #97459 -- is the redis client library faking connection for
	// the down server?
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	conn.Close()

	return redis.NewTCPClient(addr, password, database), nil
}

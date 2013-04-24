package confdis

import (
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
	c, err := New("localhost:6379", "test:confdis:simple", SampleConfig{})
	if err != nil {
		t.Fatal(err)
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

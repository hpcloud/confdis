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

func TestSimple(t *testing.T) {
	c, err := New("localhost:6379", "test:confdis:simple", SampleConfig{})
	if err != nil {
		t.Fatal(err)
	}
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
	// Seed data, using the first client
	c, err := New(
		"localhost:6379",
		"test:confdis:notify",
		SampleConfig{})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for err := range c.Changes {
			if err != nil {
				t.Fatal(err)
			}
		}
	}()
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
	// Allow reasonable delay for network/redis latency
	time.Sleep(time.Duration(100 * time.Millisecond))

	// Second client
	c2, err := New(
		"localhost:6379",
		"test:confdis:notify",
		SampleConfig{})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for err := range c2.Changes {
			if err != nil {
				t.Fatal(err)
			}
		}
	}()
	config2 := c2.Config.(*SampleConfig)

	if config2.Meta.Researcher != "Jane Goodall" {
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

	// Allow reasonable delay for network/redis latency
	time.Sleep(time.Duration(100 * time.Millisecond))

	// Second client must get notified of that change
	config2 = c2.Config.(*SampleConfig)
	if config2.Meta.Researcher != "Francine Patterson" {
		t.Fatal("did not receive change")
	}
}

// confdis manages JSON based configuration in redis
package confdis

import (
	"encoding/json"
	"github.com/vmihailenco/redis"
	"net"
)

type ConfDis struct {
	rootKey      string
	pubChannel   string
	configStruct interface{}
	redis        *redis.Client
	Changes      chan error // Channel to receive config updates (value is return of reload())
}

func New(addr, rootKey string, struc interface{}) (*ConfDis, error) {
	c := ConfDis{rootKey, rootKey + ":_changes", struc, nil, make(chan error)}
	if err := c.connect(addr); err != nil {
		return nil, err
	}
	if err := c.reload(); err != nil {
		return nil, err
	}
	return &c, c.watch()
}

// Save saves current config onto redis. WARNING: Save() may not work
// correctly if there are concurrent changes from other clients
// (notified via pubsub).
func (c *ConfDis) Save() error {
	if data, err := json.Marshal(c.configStruct); err != nil {
		return err
	} else {
		if r := c.redis.Set(c.rootKey, string(data)); r.Err() != nil {
			return r.Err()
		}
		if r := c.redis.Publish(c.pubChannel, "confdis"); r.Err() != nil {
			return r.Err()
		}
	}
	return nil
}

func (c *ConfDis) connect(addr string) error {
	// Bug #97459 -- is the redis client library faking connection for
	// the down server?
	if conn, err := net.Dial("tcp", addr); err != nil {
		return err
	} else {
		conn.Close()
	}

	c.redis = redis.NewTCPClient(addr, "", 0)
	return nil
}

// reload reloads the config tree from redis.
func (c *ConfDis) reload() error {
	// FIXME: must zero-value c.configStruct before overwriting it.
	if r := c.redis.Get(c.rootKey); r.Err() != nil {
		return r.Err()
	} else {
		return json.Unmarshal([]byte(r.Val()), c.configStruct)
	}
	panic("unreachable")
}

// watch watches for changes from other clients
func (c *ConfDis) watch() error {
	pubsub, err := c.redis.PubSubClient()
	if err != nil {
		return err
	}
	defer pubsub.Close()

	ch, err := pubsub.Subscribe(c.pubChannel)
	if err != nil {
		return err
	}

	go func() {
		for {
			<-ch
			c.Changes <- c.reload()
		}
	}()

	return nil
}

// load config from redis into etcd

package main

import (
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"log"
	"os/exec"
	"strings"
	"time"
)

func getConfigComponents() []string {
	cmd := "kato config get  | grep \"^[a-z].*\\:\" | sed -E \"s/\\://\""
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		panic(err)
	} else {
		parts := strings.Split(string(out), "\n")
		components := []string{}
		for _, part := range parts {
			if part != "node" && part != "" {
				components = append(components, part)
			}
		}
		return components
	}
}

func getAllComponentConfig() map[string]string {
	m := make(map[string]string)
	for _, component := range getConfigComponents() {
		log.Printf("Reading %s config from redis\n", component)
		out, err := exec.Command(
			"kato", "config", "get", component, "--json").Output()
		if err != nil {
			panic(err)
		}
		m[component] = string(out)
	}
	return m
}

func main() {
	// need microsecond precision to measure etcd performance.
	log.SetFlags(log.Flags() | log.Lmicroseconds)

	c := etcd.NewClient()
	allConfig := getAllComponentConfig()

	// repeat to get an idea of how long it takes to save to etcd.
	for attempt := 1; attempt <= 20; attempt++ {
		startTime := time.Now()
		for component, config := range allConfig {
			// log.Printf("Saving %s into etcd\n", component)
			_, err := c.Set(fmt.Sprintf("/config/%s", component), config, 0)
			if err != nil {
				panic(err)
			}
		}
		duration := time.Now().Sub(startTime)
		log.Printf("#%d: Took %v to save config for %d components into etcd.\n",
			attempt, duration, len(allConfig))
	}
}

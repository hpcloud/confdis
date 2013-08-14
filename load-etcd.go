// load logyard and related config into etcd

package main

import (
    "github.com/coreos/go-etcd/etcd"
    "os/exec"
	"fmt"
)

func main() {
    c := etcd.NewClient()
	for _, name := range []string{
		"logyard", "apptail", "systail", "cloud_events"} {
		fmt.Printf("Loading %s\n", name)
		out, err := exec.Command(
			"kato", "config", "get", name, "--json").Output()
		if err != nil {
			panic(err)
		}
		_, err = c.Set(fmt.Sprintf("/config/%s", name), string(out), 0)
		if err != nil {
			panic(err)
		}
	}
}

package chain

import (
	"flag"
	"fmt"
	"strings"
)

var (
	nodeName string
)

func SetNodeName(name string) {
	//lib.Log("Start node with name %q and cookie %q", name, cookie)
	if len(strings.Split(name, "@")) != 2 {
		panic(fmt.Errorf("incorrect FQDN node name (example: node@localhost)"))
	}

	nodeName = name
}

func NodeName() string {
	return nodeName
}

func init() {
	flag.StringVar(&nodeName, "name", "syntax@127.0.0.1", "node name")
}

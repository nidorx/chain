package chain

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

var (
	nodeName           string
	ErrInvalidNodeName = errors.New("incorrect FQDN node name (example: node@localhost)")
)

func SetNodeName(name string) error {
	//lib.Log("Start node with name %q and cookie %q", name, cookie)
	if len(strings.Split(name, "@")) != 2 {
		return fmt.Errorf("%w: got %q", ErrInvalidNodeName, name)
	}

	nodeName = name
	return nil
}

func NodeName() string {
	return nodeName
}

func init() {
	flag.StringVar(&nodeName, "name", "syntax@127.0.0.1", "node name")
}

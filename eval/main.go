// eval contains a tool for evaluating the scuttlebutt protocol and
// implementation.
package main

import (
	"math/rand"
	"time"

	"github.com/andydunstall/scuttlebutt/eval/cmd"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	cmd.Execute()
}

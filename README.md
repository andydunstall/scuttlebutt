# Scuttlebutt

> :warning: **Still in development.**

A Go library that facilitates cluster membership and state propagation using the
[Scuttlebutt](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf)
protocol.

The Scuttlebutt protocol is an eventually consistent, gossip-based anti-entropy
mechanism, as described in [this paper](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf).

The state of a node is represented by a set of application-defined key-value
pairs, including the node type, status, and routing information.

The state of each node is disseminated to all other nodes in the cluster,
allowing each node to develop an eventually consistent view of the entire
cluster. This state is presented as a key-value store on each node, but
typically, applications subscribe to updates about node join/leave and state
changes.

Failed nodes in the cluster are detected using the phi accrual failure detector.

The implementation is described in [docs/](docs/).

## Usage
The full API docs can be viewed with `go doc --all`.

### Create a gossip node
Creates a new node, which will start listening for updates from nodes in the
cluster. If `SeedCB` is given it will attempt to join the cluster by gossiping
with these nodes.

Whenever the node doesn't know about any other peers it will re-seed by calling
`SeedCB` to get a new list of seeds.

```go
node := scuttlebutt.Create(
	"0.0.0.0:8229",
	scuttlebutt.WithSeedCB(func() []string {
		return myconfig.Seeds
	}),
	scuttlebutt.WithOnJoin(...),
	scuttlebutt.WithOnLeave(...),
	scuttlebutt.WithOnUpdate(...),
)
```

See [`options.go`](options.go) for the full set of options.

### Update our nodes state
Updates our nodes local state, which will be propagated to other nodes in the
cluster and notify their subscribes of the update.

```go
node.UpdateLocal("routing.addr", "10.25.104.42:5112")
node.UpdateLocal("state", "ready")
```

Note the keys and values are limitted to 256 bytes.

### Lookup the known state of another node
Looks up the state of the peer as known by this node. Since the cluster
membership is eventually consistent this may be out of date with the actual
peer, though should converge quickly.

Note typically you'll subscribe to updates with `OnUpdate` rather than querying
directly.

```go
addr, ok := node.Lookup("10.26.104.82:7188", "routing.addr")
if !ok {
	// ...
}
```

## Building
Assuming you have Go installed, simply build with
```bash
$ go build
```

### API Docs
Show the API docs with
```bash
$ go doc --all
```

### Testing
Tests are split into unit and system tests.

Unit tests test small units in isolation, with no goroutines or networking, so
should finish very fast. These sit alongside the code under test so can be
ran with
```bash
$ go test
```

System tests spin up a cluster of nodes to test. These are kept in the `tests/`
directory so can be run with
```bash
$ go test ./...
```

### Evaluation
Theres a CLI tool in `eval/` that can be used to evaluate the cluster. Such
as the time it takes to propagate an update to all nodes in a cluster with
64 nodes.

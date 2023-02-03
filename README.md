# Scuttlebutt
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

The implementation is described in [docs/](docs/).

**WIP**
* Working on adding the failure detector described in [docs/](docs/)

## Usage
The full API docs can be viewed with `go doc --all`.

### Create a gossip node
Creates a new node, which will start listening for updates from nodes in the
cluster. If `SeedCB` is given it will attempt to join the cluster by gossiping
with these nodes. Note whenever the node doesn't know about any other peers it
will re-seed by calling `SeedCB` to get a new list of seeds.

The size of the node ID must not exceed 256 bytes.

```go
node := scuttlebutt.Create(&scuttlebutt.Config{
	ID: "773dc6df",
	BindAddr: "0.0.0.0:8229",
	SeedCB: func() []string {
		return myconfig.Seeds
	},
	// Receive events about nodes joining, leaving and updating.
	OnJoin: func(peerID string) {
		fmt.Println("Node joined", peerID)
	},
	OnLeave: func(peerID string) {
		fmt.Println("Node joined", peerID)
	},
	OnUpdate: func(peerID string, key string, value string) {
		fmt.Println("Node updated", peerID, key, value)
	}
})
```

### Update our nodes state
Updates our nodes local state, which will be propagated to other nodes in the
cluster and notify their subscribes of the update.

```go
node.UpdateLocal("routing.addr", "10.25.104.42:5112")
node.UpdateLocal("state", "ready")
```

### Lookup the known state of another node
Looks up the state of the peer as known by this node. Since the cluster
membership is eventually consistent this may be out of date with the actual
peer, though should converge quickly.

Note typically you'll subscribe to updates with `OnUpdate` rather than querying
directly.

```go
addr, ok := node.Lookup("9a023689", "routing.addr")
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

## Future Improvements

### Configuration
Adding default configuration would be useful, similar to memberlists `DefaultLAN`
and `DefaultWAN` config.

Also being able to configure the initial state for the node before it joins the
cluster.

### Failure Detector
Currently its left to the application to detect failed nodes, though this could
be done within the library itself.

Similarly theres no way for a node to explicitly leave the cluster.

### Evaluation
Evaluating how nodes behave under different loads and chaos would be useful,
and doing some CPU profilings to look for any optimisations.

Could use something like [toxiproxy](https://github.com/Shopify/toxiproxy)
to inject faults.

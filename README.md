# Scuttlebutt

A Go library that manages cluster membership and propagates node state using
the [Scuttlebutt](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf)
protocol.

Scuttlebutt is an eventually consistent anti-entropy protocol based on gossip,
described in [this paper](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf).

The state of each node is simply a set of arbitrary key-value pairs set by the
application. Such as it may include the node type, the node status and routing
information for the node.

Each nodes state is propagated to all other nodes in the cluster, so a node
builds an eventually consistent view of the cluster. This state is exposed
as a key-value store on each node, though typically apps will subscribe to
updates about nodes joining, leaving and updating their state.

The implementation is described in [docs/](docs/).

**Note** this does not currently support detecting nodes leaving the cluster.
Working on adding a [Phi Accrual Failure Detector](https://www.computer.org/csdl/proceedings-article/srds/2004/22390066/12OmNvT2phv)
similar to Cassandra to detect failed nodes. Though even with the failure
detector, apps may choose to also explicitly signal to other nodes that a node
is leaving though the nodes state.

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
This is only a fairly simple implementation so far, which is functional though
theres lots that could be done to improve:

### Limit Messages to MTU
Currently the protocol uses UDP but has no limits on the packet size. To support
this:
* At the moment assume if a peer isn't in a digest, the sender doesn't know
about that peer, though if we're limiting what can go in the digest this will
no longer be the case. This should be fine as if the sender doesn't know about
the peer, it will learn about it when the receiver responds with it's own
digest (though may require another gossip round)
* Limit the size of digests to fit in the MTU, either by randomly selecting
a subset of peers to include, or sending a digest split over multiple messages
each gossip round
* Limit the size of the deltas to fit in the MTU, which the paper recommends
including the most out of date deltas relative to the requested digest (by
comparing versions)

### Binary Protocol
At the moment everything is encoded with JSON which is not very efficient (both
in time to encode and space). Given payloads are quite simple a binary protocol
that includes a header with the message type and a sequence of digest/delta
entries should be easy and efficient.

This will also help limitting messages to the MTU as we can just keep adding
digests/deltas (ordered by preference) to the message until adding another would
exceed the limit.

### Configuration
Adding default configuration would be useful, similar to memberlists `DefaultLAN`
and `DefaultWAN` config.

Also being able to configure the initial state for the node before it joins the
cluster.

### Phi-accrual failure detector
Currently its left to the application to detect failed nodes, though this could
be done within the library itself.

Similarly theres no way for a node to explicitly leave the cluster.

### Evaluation
Evaluating how nodes behave under different loads and chaos would be useful,
and doing some CPU profilings to look for any optimisations.

Could use something like [toxiproxy](https://github.com/Shopify/toxiproxy)
to inject faults.

# Scuttlebutt
**In progress:** See TODO below.

A Go library that manages cluster membership and propagates node state using
the [Scuttlebutt](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf)
protocol.

Scuttlebutt is an eventually consistent anti-entropy protocol based on gossip,
described in the paper [here](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf).

The state of each node is simply a set of arbitrary key-value pairs, so its
upto the application what state is needed. Such as it may include the node
type, the nodes current state and any networking information needed to route
to the node.

Each nodes state is propagated to all other nodes in the cluster, so each node
builds an eventually consistent store containing its known state about each
other node. The application can subscribe to updates and lookup key-value
pairs for a given node.

## Usage
The full API docs can be viewed with `go doc --all`.

**Create a gossip node**

Creates a new node, which will start listening for updates from nodes in the
cluster. Since it does not yet know about any other nodes, it will not join
the cluster unless contacted by another node.

```go
node := scuttlebutt.Create(&scuttlebutt.Config{
	ID: "node-1",
	BindAddr: "0.0.0.0:8119",
	// Subscribe to events about nodes joining and leaving.
	NodeSubscriber: mySubscriber,
	// Subscribe to state updates from other nodes.
	StateSubscriber: mySubscriber,
})
```

**Join the cluster**

To join the cluster we must tell the node the address of at least one other
node in the cluster. Once it syncs with these nodes it will learn about other
nodes in the cluster and contact them directly in the future.

```go
node.Seed([]string{"10.26.104.52:9992", "10.26.104.56:7331"})
```

**Update our nodes state**

Updates our nodes local state, which will be propagated to other nodes in the
cluster and notify their subscribes of the update.

```go
node.UpdateLocal("rpcAddr", "10.25.104.42:5112")
node.UpdateLocal("state", "ready")
```

**Lookup the known state of another node**

Looks up the state of the peer as known by this node. Since the cluster
membership is eventually consistent this may be out of date with the actual
peer, though should converge quickly.

Note typically you'll subscribe to updates with `Config.StateSubscriber`
rather than querying directly.

```go
addr, ok := node.Lookup("peer-2", "rpcAddr")
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

## TODO
The basic implementation does now work, though is only a simplified version of
Scuttlebutt so still needs work.
- [ ] Push/pull
- [ ] Add configurable initial state to avoid sending an empty node state when
first joining
- [ ] Limit digests and deltas size to MTU
	* When deltas exceeds the MTU - send the deltas with the most entries to
send the oldest entries first
- [ ] Protocol: currently only a simplified version of Scuttlebutt, needs
extending to match the protocol described in the paper
- [ ] Binary protocol
- [ ] API docs
- [ ] Add phi-accrual failure detector
- [ ] Config
	* Max entries per delta: used to limit the rate gossips receive entries
- [ ] Add CI
- [ ] Load test evaluation
	* Look at reproducing the papers results
- [ ] Testing
	* Add chaos testing using fake transport and injecting partitions

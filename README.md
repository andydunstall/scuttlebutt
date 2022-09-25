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
```go
node := scuttlebutt.Create(&scuttlebutt.Config{
	ID: "node-1",
	BindAddr: "0.0.0.0:8119",
	// Subscribe to updates about other nodes.
	EventSubscriber: sub,
})

// Set this nodes state to be propagated to other nodes when joining.
node.UpdateLocal("service", "ordering")
node.UpdateLocal("rpcAddr", "10.25.104.42:5112")
node.UpdateLocal("state", "booting")

// Join an existing cluster by specifying at least on known peer.
node.Join([]string{"10.26.104.52:9992", "10.26.104.56:7331"})

// ...

// Update this nodes state, which will be propagated to other nodes (and notify
// subscribers of those nodes).
node.UpdateLocal("state", "active")

// ...
```

## Building
Assuming you have Go installed, simply build with
```bash
$ go build
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
- [ ] Limit digests and deltas size to MTU
- [ ] Protocol: currently only a simplified version of Scuttlebutt, needs
extending to match the protocol described in the paper
- [ ] Binary protocol
- [ ] API docs
- [ ] Add phi-accrual failure detector

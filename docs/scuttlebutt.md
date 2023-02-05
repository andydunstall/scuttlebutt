# Scuttlebutt
This document gives an overview of how Scuttlebutt is implemented, though
does not give an in depth description of the protocol since that is covered in
the [paper](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf).

## Peer State
Each node contains an in-memory store containing its known state about the
all peers in the cluster (including itself).

This state contains:
* The peer address,
* A set of versioned key-value pairs (containing the application state),
* The peers version as known by this node, which is the maximum version of
the peers key-value pairs.

Such as an entry for a known peer may have state:
```
{
	"addr": "10.26.104.64:8119",
	"version": 14,
	"state": {
		# key: (value, version)
		"status": ("booting", 10),
		"rpc.addr": ("10.26.104.64:7138", 14),
		"type": ("router", 11),
		"id": ("b0a09141", 2)
	}
}
```

The peers address is used to uniquely identify the peer.

This view of the peers in the cluster is eventually consistent (excluding the
nodes known state about itself, which will always be the latest version given
nodes can only update their own state).

## Update State
A node can only update its own key-value pairs. Such as if the above peer is
updated with `status=active`, that peers version is incremented to `15` and
set as the version for the new entry. This version is used by to compare
the known versions of a peer when gossiping, and quickly decide which is more
up to date.

So the entry becomes:
```
{
	"addr": "10.26.104.64:8119",
	"version": 15,
	"state": {
		# key: (value, version)
		"status": ("active", 15),
		"rpc.addr": ("10.26.104.64:7138", 14),
		"type": ("router", 11)
		"id": ("b0a09141", 2)
	}
}
```

Peers with no key-value pairs start with a version of 0.

## Gossip
Each node initiates a round of gossip at a configured rate.

Each around the node:
1. Chooses a random alive node from its set of known alive peers and sends
it a digest request (described below),
  a. If there are no known alive nodes, re-seeds by sending a digest request
to all seed addresses,
2. Checks the liveness of each known node using the [failure detector](./failure-detector.md):
  a. If the node has gone down its state is updated and the application is
notified about the node leaving,
  b. If the node was previously down and has come back up the state is updated
and the application is notified about the node re-joining,
3. Chooses a random down node (if any) and sends a digest request. This is to
check for nodes coming back up,
  * Once a node has been down for an hour it is removed

### Send Digest Request
Node A requests any state that node B has that it doesn't by sending a
digest request.

Node A fetches a shuffled list of peers from its known state and iterates. For
each peer the peers address and version to the digest. This does not include
the key-value state for that peer.

To avoid exceeding the configured maximum payload size, it stops adding entries
to the digest once the payload is full (and adding any more entries would
exceed the limit). So if the cluster is large the digest may not contain all our
known peers, though since entries are chosen at random, any missed entries will
eventually be included in a future round.

### Receive Digest Request
Node B receives the digest request, applies it to its local state, then responds
with delta and digest responses.

#### Apply Digest
Node B will receive the digest request and check if there are any peers in
the request that it doesn't know about. If there is it will add the peer to
its known state with a version of 0 (we it doesn't have any key-value pairs for
the node). Node B then sends a delta response and a digest response.

#### Delta Response
The delta response contains any state node B knows about that node A doesn't.

To avoid exceeding the configured maximum payload size we add state in order
or how out of date the sender is and stop adding entries once the response
is full.

If the delta response is empty we don't send it. Its use used for the
liveness check by the failure detector so theres no need.

#### Digest Response
In addition to the delta response, the node B then requests any state that node
A has that it doesn't in a digest response.

The digest response is always sent even if it is empty since its used as
heartbeats by the failure detector.

## Receive Delta Response

### Apply Delta
When node A receives the delta response it iterates though all key-value-version
tuples for the peers and update its local state. It must only update
a key-value pair if the version in the delta exceeds its known version. Its
possible node A received a more up to date version since sending the digest and
receiving the delta response.

If the key-value version is greater than the peer version, the peer version is
set to this larger version. This means the peers version is always the same
as the largest key-value version.

## Receive Digest Response
The digest response is handled the same as a digest request, except it doesn't
respond with its own digest.

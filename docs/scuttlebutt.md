# Scuttlebutt
This document describes how the library implements the Scuttlebutt protocol. It
does not describe how Scuttlebutt works since that is covered in the [paper](https://www.cs.cornell.edu/home/rvr/papers/flowgossip.pdf).

## State
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
		"type": ("router", 11)
	}
}
```

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
	}
}
```

Peers with no key-value pairs start with a version of 0.

## Gossip
Each node periodically initiates a round of gossip at a configured rate. A round
means the node chooses another node to gossip with and they exchange any state
about the cluster that the other is missing. So once the round is complete both
nodes should have the same view of the cluster.

A gossip round on node A consists of:
1. Chooses a random node, B, from its set of known peers (excluding itself),
2. Sends a digest request containing A's set of known peers and their
versions (excluding the peers key-value pairs) to node B,
3. Node B checks A knows about any peers it doesn't from the digest. If it does
it stores those peers with a version of 0 (as it doesn't have any state for
the peer yet),
4. Node B then compares each peer version in the digest with its own known state
about the peer, so either:
  * Its known version is greater than the version in the digest, meaning node B
knows more up to date state about the peer than node A,
  * Its known version is less than the version in the digest, meaning node A
knows more up to date state about the peer than node B,
  * The versions are the same so they both know the same about the peer,
5. Node B sends a delta response containing any key-value pairs that node A is
missing about node B's known set of peers,
6. Node B also sends a digest response containing only the peers and versions it
knows node A has but it doesn't (after comparing node A's digest),
7. Node A receives the delta response and updates its local state,
8. Node B also receives a digest response which it handles the same way as B,
except it doesn't respond with its own digest.

Alternatively, if the node does not know about any other peers (since it just
started), it will instead initiate a round of gossip with all configured
seeds to try and join the cluster as fast as possible.

This means whenever a node updates its local state, such as in the above example
where peer `peer_9873` updates its state so its version increases to `15`. When
node `peer_9873` gossips with another node its version about `peer_9873` will
be greater than that of the remote node so will send it the latest version. That
remote node will then do its own round of gossip with another node and send it
the latest version of peer `peer_9873`. So the update quickly propagates around
the cluster.

After node A has gossiped with a peer, it will check if there are any seed nodes
it does not know about it. If so it will select one at random and send it
a digest.

Finally, the node checks the suspision level of each node. If any nodes are
critical it sends a digest to those nodes. The status of each node is also
updated (up or down).

## Digest Request
Node A requests any state that node B has that it doesn't by sending a
digest request.

Node A fetches a shuffled list of peers from its known state and iterates. For
each peer it adds to the digest the:
* Peer address,
* Peer version.
Note this does not include the key-value state for that peer.

To avoid exceeding the configured maximum payload size, it stops adding entries
to the digest once the payload is full (and adding any more entries would
exceed the limit). So if the cluster is large the digest may not contain all our
known peers, though since entries are chosen at random, any missed entries will
eventually be included in a future round.

## Delta Response
The delta response contains any state we know about that the digests sender
doesn't.

To avoid exceeding the configured maximum payload size we add state in order
or how out of date the sender is and stop adding entries once the responds
is full.

This means we start by iterating though each entry in the digest and comparing
the digests known version of a peer and our known version of that peer. If
our version is greater than the digest version we know we have some state that
the sender doesn't, so build a list of peer address that the sender is missing state
on, sorted by the difference in versions. This means the first peer in the list
is the peer the sender is missing the most state on.

We then iterate this list, and for each peer add key-value-version tuples
for any key-value pairs whos version is greater than that in the digest. Note
these entries for peer must be sorted by version as if we can only send a subset
of pairs we miss versions. Once the message is full we stop.

Note we send the delta response even if it is empty (when we don't have any
state the sender doesn't) since it will be used for a liveness check by the
failure detector and calculate the RTT. Otherwise we may end up never responding
so being considered dead.

Once the delta is sent, if we found there are entries the sender has but we
don't (when the digest version exceeded our known version for a peer), we can
include these in our own digest response and send that to the peer. If this is
empty theres no need to send anything (we already send a delta response even if
it is empty so thats enough for the failure detector).

Note since the original delta request doesn't nessesarily contain all entries
we may still be missing state that we didn't include in the digest responds,
though will get that when we start our own gossip round and send a full digest
request. The response is just to get any state that we can be sure we missed.

## Receive Delta Response and Digest Response
When we receive a delta responds we can iterate though all key-value-version
tuples for the peers and update out local state. Note we must only update
a key-value pair if the version in the delta exceeds our known version. Its
possible we've received a more up to date version since sending the digest and
receiving a delta responds.

If the key-value version is greater than the peer version, the peer version is
sete to this larger version. This means the peers version is always the same
as the largest key-value version.

We may also receive a digest responds, which we respond too the same as
a digest request, except don't send another digest response.

## Codec
Each message is prefixed with a 1 byte (`uint8`) type:
* `DIGEST-REQUEST`: `1`
* `DIGEST-RESPONSE`: `2`
* `DELTA`: `3`

Since only UDP is supported no framing information is needed.

Variable size strings (such as the peer address) are prefixed with
their `uint8` size.

### `DIGEST-REQUEST`
Contains a list of entries appended together, each containing:
* Peer address: Encoded string (note we encode as a string rather than integer
format as support custom transports where the address format is unknown),
* Peer version: `uint64`

### `DIGEST-RESPONSE`
This is the same format as `DIGEST-REQUEST` except has a different type to
indicate the receiver should not respond with its own digest.

### `DELTA`
Contains a list of entries appended together, each containing:
* Peer address: Encoded string,
* Key: Encoded string,
* Value: Encoded string,
* Version: `uint64`

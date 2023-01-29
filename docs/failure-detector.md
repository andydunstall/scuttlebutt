# Failure Detector
Each node runs a Phi Accrual Failure Detector to make a decision on whether
its known peers are live or not.

The failure detector maintains a sliding window holding arrival times of past
heartbeats for each peer.

Digests are used as heartbeats since these are expected to be exchanged between
peers at regular intervals. Note not including deltas since they are only sent
when they are non-empty, though deltas will always be sent. Digests are sent
by peers both as part of a gossip round, and in response to a digest we send to
them.

The failure detector outputs a suspision level (phi) for each known peer. The
higher the suspision level, this higher chance there is that peer is down.

It is configured with two thresholds:
* Critical threshold: If exceeded the node is considered critical so tries to
gossip with it immediately,
* Down threshold: If exceeded the node is considered to be down so the
application is notified.

Each gossip round nodes recalculate the suspision level for each known peer. If
that suspision level exceeds the cricical threshold, it will immediately send a
digest to the peer. If it exceeds the down threshold the peer is marked as
down and the application is notified.

After a peer is considered down, nodes will periodically try to gossip to that
node. Once a peer is considered down for the expiry timeout it will be removed
and so nodes stop trying it.

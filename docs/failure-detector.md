# Failure Detector
Each node runs a Phi Accrual Failure Detector to make a decision on whether
its known peers are up or down.

The failure detector maintains a sliding window holding arrival times of past
heartbeats for each peer.

Digests are used as heartbeats since these are expected to be exchanged between
peers at regular intervals. Note not including deltas since they are only sent
when they are non-empty, though deltas will always be sent. Digests are sent
by peers both as part of a gossip round, and in response to a digest we send to
them.

The failure detector outputs a suspision level (phi) for each known peer. The
higher the suspision level, this higher chance there is that peer is down. If
the suspision level exceeds the configured conviction threshold is is considered
down.

Each round the gossiper will try to gossip with a down node to check if it has
come back up.

Once a peer is considered down for an hour it will be removed and will stop
trying it.

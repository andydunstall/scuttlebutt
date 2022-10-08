package scuttlebutt

type NodeSubscriber interface {
	// NotifyJoin is invoked when a node is detected to have joined.
	// The Node argument must not be modified.
	NotifyJoin(peerID string)
}

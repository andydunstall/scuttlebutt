package scuttlebutt

type StateSubscriber interface {
	// NotifyUpdate is invoked when a peers entry is updated.
	NotifyUpdate(peerID string, key string, value string)
}

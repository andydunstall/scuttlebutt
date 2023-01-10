package foo

type foo struct {
}

func (f *foo) Gossip(peerID string, addr string) {
	peers := peermap.PeerIDs()
	rand.shuffle(peers)

	m := []byte{}
	m.push(TypeDigestRequest)
	for peerid in peers
	  digest = peermap.digest(peerid)
	  digestEnc = encodedigest(digest)
	  if len(m) + len(digest) > max payload size
	    break
	  m.push(digest)

	f.transport.Send(addr, digest)
}

func (f *foo) Seed() {
	if g.seedCB == nil {
		g.logger.Debug("no seed cb; skipping")
		return
	}

	seeds := g.seedCB()

	g.logger.Debug("seeding gossiper", zap.Strings("seeds", seeds))

	for _, addr := range seeds {
		// Ignore ourselves.
		if addr == g.BindAddr() {
			continue
		}
		g.gossip("seed", addr)
	}
}

func (f *foo) OnMessage(b []byte) {
	type = b[0]

	switch type
	  digest request => onDigestRequest(decodeDigest(...))
	  digest responds => onDigestResponds(decodeDigest(...))
	  delta => onDelta(decodeDelta(...))
}

func (f *foo) onDigestRequest(req []digest) {
	// any peers in digest not local just add locally

	peersByVersionDiff = ...
	// iterate in order of largest version diff
	deltaResp = []byte{}
	deltaResp.push(TypeDelta)
	for peer in peersByVersionDiff)
	  for deltaEntry in peermap.Deltas(peerid, digestEntry)
	    deltaEnctyEnc = encodeDeltaEntry
		if len(...) > max payload size
		  break
		deltaResp.push

	f.transport.Send(addr, digest)

	digestResp = []byte
	digestResp.push(typeDigestResponds)
	for revers iter peersByVersionDiff
	  if digestEntry.version > local version
	    // add to digest resp

	f.transport.Send(..., digestResp)
}

func (f *foo) onDigestResponds(resp []digest) {
	same as onDigestRequest except dont send digest resp
}

func (f *foo) onDelta(resp delta) {
	for entry in delta
	  f.peermap.updateRemote(entry.peerID, ...)
}

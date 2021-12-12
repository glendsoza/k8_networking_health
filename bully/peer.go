package bully

// Peer is a `struct` representing a remote `bully.Bully`.
type Peer struct {
	ID    string
	addr  string
	alive bool
}

type PeerInfo struct {
	ID    string `json:"id"`
	Addr  string `json:"address"`
	Alive bool   `json:"alive"`
}

// NewPeer returns a new `*bully.Peer`.
func NewPeer(ID, addr string) *Peer {
	return &Peer{ID: ID, addr: addr, alive: true}
}

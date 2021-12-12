package bully

import (
	"sync"
)

// Peers is an `interface` exposing methods to handle communication with other
// `bully.Bully`s.
//
// NOTE: This project offers a default implementation of the `Peers` interface
// that provides basic functions. This will work for the most simple of use
// cases fo exemples, although I strongly recommend you provide your own, safer
// implementation while doing real work.
type Peers interface {
	Add(ID, addr string)
	Delete(ID string)
	Find(ID string) bool
	UpdateStatus(ID string, status bool)
	GetStatus(ID string) bool
	DeleteAll()
	PeerData() []PeerInfo
}

// PeerMap is a `struct` implementing the `Peers` interface and representing
// a container of `bully.Peer`s.
type PeerMap struct {
	mu    *sync.RWMutex
	peers map[string]*Peer
}

// NewPeerMap returns a new `bully.PeerMap`.
func NewPeerMap() *PeerMap {
	return &PeerMap{mu: &sync.RWMutex{}, peers: make(map[string]*Peer)}
}

// Add creates a new `bully.Peer` and adds it to `pm.peers` using `ID` as a key.
//
// NOTE: This function is thread-safe.
func (pm *PeerMap) Add(ID, addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.peers[ID] = NewPeer(ID, addr)
}

// Delete erases the `bully.Peer` corresponding to `ID` from `pm.peers`.
//
// NOTE: This function is thread-safe.
func (pm *PeerMap) Delete(ID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.peers, ID)
}

// Find returns `true` if `pm.peers[ID]` exists, `false` otherwise.
//
// NOTE: This function is thread-safe.
func (pm *PeerMap) Find(ID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, ok := pm.peers[ID]
	return ok
}

// Write writes `msg` to `pm.peers[ID]`. It returns `nil` or an `error` if
// something occurs.
//
// NOTE: This function is thread-safe.

func (pm *PeerMap) UpdateStatus(ID string, status bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.peers[ID].alive = status
}

func (pm *PeerMap) GetStatus(ID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.peers[ID].alive
}

// PeerData returns a slice of anonymous structures representing a tupple
// composed of a `Peer.ID` and `Peer.addr`.
//
// NOTE: This function is thread-safe.
func (pm *PeerMap) PeerData() []PeerInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	var IDSlice []PeerInfo
	for _, peer := range pm.peers {
		IDSlice = append(IDSlice, PeerInfo{peer.ID, peer.addr, peer.alive})
	}
	return IDSlice
}

func (pm *PeerMap) DeleteAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for id, p := range pm.peers {
		log.Debug().
			Str("err", "").
			Str("id", "").
			Str("coordinator", "").
			Str("address", "").
			Msgf("Deleted peer %s", p.addr)
		delete(pm.peers, id)
	}
}

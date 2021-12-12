package bully

import (
	"encoding/json"
	"fmt"
	"knh/utils"
	"net/http"
	"os"

	"strings"
	"sync"
	"time"
)

var log = utils.GetLogger()
var connectCooldownPeriod = utils.ParseEnvElseDefault("CONNECT_COOLDOWN_PERIOD", 2)
var connectMaxRetries = utils.ParseEnvElseDefault("CONNECT_MAX_RETRIES", 5)
var sendMaxRetries = utils.ParseEnvElseDefault("SEND_MAX_RETRIES", 5)
var sendCooldownPeriod = utils.ParseEnvElseDefault("SEND_COOLDOWN_PERIOD", 1)
var electionCooldownPeriod = utils.ParseEnvElseDefault("ELECTION_COOLDOWN_PERIOD", 15)
var httpClient = &http.Client{Timeout: time.Second * 5}

// Sanity check url
var sanityCheckURL = os.Getenv("SANITY_CHECK_URL")

//Status url should accept post request
var peerStatusURL = os.Getenv("PEER_STATUS_URL")
var clusterStatusURL = os.Getenv("CLUSTER_STATUS_URL")

// Bully is a `struct` representing a single node used by the `Bully Algorithm`.
//
// NOTE: More details about the `Bully algorithm` can be found here
// https://en.wikipedia.org/wiki/Bully_algorithm .

type Bully struct {
	ID           string
	addr         string
	coordinator  string
	peers        Peers
	mu           *sync.RWMutex
	receiveChan  chan Message
	electionChan chan Message
}

// NewBully returns a new `Bully` or an `error`.
//
// NOTE: All connections to `Peer`s are established during this function.
func NewBully(ID, addr, proto string, peers map[string]string) (*Bully, error) {
	b := &Bully{
		ID:           ID,
		addr:         addr,
		coordinator:  ID,
		peers:        NewPeerMap(),
		mu:           &sync.RWMutex{},
		electionChan: make(chan Message, 1),
		receiveChan:  make(chan Message),
	}
	go func() {
		http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("pong"))
		})

		http.HandleFunc("/coordinator", func(w http.ResponseWriter, req *http.Request) {
			b.SetCoordinator(req.URL.Query().Get("id"))
		})
		err := http.ListenAndServe(b.addr, nil)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("id", "").
				Str("coordinator", "").
				Str("address", "").
				Msg("Failed to Listen")
		}
	}()
	b.Connect(proto, peers)
	return b, nil
}

func (b *Bully) GetAddress() string {
	return b.addr
}

// Connect performs a connection to the remote `Peer`s.
func (b *Bully) Connect(proto string, peers map[string]string) {
	// Delete the existing peers if they are present
	b.peers.DeleteAll()
	for ID, addr := range peers {
		if b.ID == ID {
			continue
		}
		log.Debug().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Trying to connect to %s", addr)
		for attempts := 1; ; attempts++ {
			_, err := http.Get(fmt.Sprintf("http://%s/ping", addr))
			if err != nil {
				log.Debug().
					Err(err).
					Str("id", b.ID).
					Str("coordinator", b.coordinator).
					Str("address", b.addr).
					Msgf("Failed to connect to %s with %d attempts", addr, attempts)
				if attempts >= connectMaxRetries {
					log.Warn().
						Err(err).
						Str("id", b.ID).
						Str("coordinator", b.coordinator).
						Str("address", b.addr).
						Msgf("Failed to connect to %s with %d attempts", addr, attempts)
					break
				}
				time.Sleep(time.Duration(connectCooldownPeriod) * time.Second)
				continue
			}
			log.Debug().
				Str("err", "").
				Str("id", b.ID).
				Str("coordinator", b.coordinator).
				Str("address", b.addr).
				Msgf("Connected to %s", addr)
			b.peers.Add(ID, addr)
			break
		}
	}
	log.Debug().
		Str("err", "").
		Str("id", b.ID).
		Str("coordinator", b.coordinator).
		Str("address", b.addr).
		Msgf("Peers are %v", b.peers.PeerData())
}

func (b *Bully) DoPostRequest(url string, data []byte) {
	resp, err := httpClient.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		log.Warn().
			Err(err).
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Failed to do post request %s", url)
	} else if resp.StatusCode != http.StatusOK {
		log.Warn().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Server responded with non 200 status code for %s", url)
	}
}

func (b *Bully) Inform(url string, data interface{}) {
	var err error
	var postData []byte
	switch d := data.(type) {
	case PeerInfo:
		postData, err = json.Marshal(struct {
			Peer               PeerInfo `json:"peer"`
			Coordinator        string   `json:"coordinator"`
			CoordinatorAddress string   `json:"coordinator_address"`
		}{
			Peer:               d,
			Coordinator:        b.ID,
			CoordinatorAddress: b.addr,
		})
	case []PeerInfo:
		postData, err = json.Marshal(struct {
			PeerMap            []PeerInfo `json:"peer_map"`
			Coordinator        string     `json:"coordinator"`
			CoordinatorAddress string     `json:"coordinator_address"`
		}{
			PeerMap:            d,
			Coordinator:        b.ID,
			CoordinatorAddress: b.addr,
		})
	default:
		log.Warn().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Post data is not of type PeerInfo or []PeerInfo")
		return
	}
	if err != nil {
		log.Warn().
			Err(err).
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msg("Failed to marshal the peer data")
		return
	}
	go b.DoPostRequest(url, postData)
}

// Marks a peer as dead and sends post request to peer status endpoint
func (b *Bully) MarkDeadAndInform(to, addr string) {
	// perform a sanity check
	_, err := http.Get(sanityCheckURL)
	if err != nil {
		log.Info().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Coordinator sanity check passed, marking peer %s as dead", to)
		// mark the peer as dead
		b.peers.UpdateStatus(to, false)
		if peerStatusURL != "" {
			b.Inform(peerStatusURL, PeerInfo{to, addr, false})
		}
	} else {
		log.Warn().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Coordinator sanity check failed")
	}
}

// Send sends a `bully.Message` of type `what` to `b.peer[to]` at the address
// `addr`. If no connection is reachable at `addr` or if `b.peer[to]` does not
// exist, the function retries five times (configurable via env variable) and returns an `error` if it does not
// succeed.
func (b *Bully) Send(to, addr string) error {

	log.Info().
		Str("err", "").
		Str("id", b.ID).
		Str("coordinator", b.coordinator).
		Str("address", b.addr).
		Msgf("Sending message to %s", to)

	for attempts := 1; ; attempts++ {
		_, err := http.Get(fmt.Sprintf("http://%s/coordinator?id=%s", addr, b.ID))
		if err != nil {
			if attempts >= sendMaxRetries {
				log.Warn().
					Err(err).
					Str("id", b.ID).
					Str("coordinator", b.coordinator).
					Str("address", b.addr).
					Msgf("Failed to send message to %s with max attempts of %d", to, attempts)
				if b.ID == b.coordinator && b.peers.GetStatus(to) {
					b.MarkDeadAndInform(to, addr)
				}
				return err
			}
			log.Info().
				Err(err).
				Str("id", b.ID).
				Str("coordinator", b.coordinator).
				Str("address", b.addr).
				Msgf("Failed to send message to %s with %d attempts", to, attempts)
			time.Sleep(time.Duration(sendCooldownPeriod) * time.Second)
		} else {
			log.Info().
				Str("err", "").
				Str("id", b.ID).
				Str("coordinator", b.coordinator).
				Str("address", b.addr).
				Msgf("Successfully sent message to %s", to)
			b.peers.UpdateStatus(to, true)
			return nil
		}
	}
}

// SetCoordinator sets `ID` as the new `b.coordinator` if `ID` is greater than
// `b.coordinator` or equal to `b.ID`.
//
// NOTE: This function is thread-safe.
func (b *Bully) SetCoordinator(ID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ID > b.coordinator || ID == b.ID {
		log.Info().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Settting %s as coordinator", ID)
		b.coordinator = ID
	}
}

// Coordinator returns `b.coordinator`.
//
// NOTE: This function is thread-safe.
func (b *Bully) Coordinator() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.coordinator
}

// Elect handles the leader election mechanism of the `Bully algorithm`.
func (b *Bully) Elect() {
	log.Info().
		Str("err", "").
		Str("id", b.ID).
		Str("coordinator", b.coordinator).
		Str("address", b.addr).
		Msgf("Electing coordinator")
	for _, rBully := range b.peers.PeerData() {
		if rBully.ID > b.ID {
			log.Debug().
				Str("err", "").
				Str("id", b.ID).
				Str("coordinator", b.coordinator).
				Str("address", b.addr).
				Msgf("Communicating with superior %s", rBully.Addr)
			_, err := http.Get(fmt.Sprintf("http://%s/ping", rBully.Addr))
			if err == nil {
				log.Debug().
					Str("err", "").
					Str("id", b.ID).
					Str("coordinator", b.coordinator).
					Str("address", b.addr).
					Msgf("Superior %s responded", rBully.Addr)
				return
			} else {
				log.Debug().
					Err(err).
					Str("id", b.ID).
					Str("coordinator", b.coordinator).
					Str("address", b.addr).
					Msgf("Superior %s did not respond", rBully.Addr)
			}
		}
	}
	// if no superior is available, set the current bully as the coordinator
	log.Info().
		Str("err", "").
		Str("id", b.ID).
		Str("coordinator", b.coordinator).
		Str("address", b.addr).
		Msgf("No superior found, setting myself as coordinator")
	b.SetCoordinator(b.ID)
	peerData := b.peers.PeerData()
	for _, rBully := range peerData {
		log.Debug().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.coordinator).
			Str("address", b.addr).
			Msgf("Informing %s about coordinator", rBully.Addr)
		_ = b.Send(rBully.ID, rBully.Addr)
	}
	//
	if clusterStatusURL != "" {
		b.Inform(clusterStatusURL, peerData)
	}
}

// Run launches the two main goroutine. The first one is tied to the
// execution of `workFunc` while the other one is the `Bully algorithm`.
//
// NOTE: This function should be an infinite loop.

func (b *Bully) Run() {
	log.Info().
		Str("err", "").
		Str("coordinator", b.Coordinator()).
		Str("address", b.GetAddress()).
		Msgf("Starting bully with CONNECT_COOLDOWN_PERIOD=%d,"+
			"CONNECT_MAX_RETRIES=%d,"+
			"SEND_COOLDOWN_PERIOD=%d,"+
			"SEND_MAX_RETRIES=%d,"+
			"ELECTION_COOLDOWN_PERIOD=%d,"+
			"PEER_STATUS_URL=%s,"+
			"CLUSTER_STATUS_URL=%s",
			connectCooldownPeriod,
			connectMaxRetries,
			sendCooldownPeriod,
			sendMaxRetries,
			electionCooldownPeriod,
			peerStatusURL,
			clusterStatusURL)
	for {
		log.Info().
			Str("err", "").
			Str("id", b.ID).
			Str("coordinator", b.Coordinator()).
			Str("address", b.GetAddress()).
			Msg("Sleeping before calling election")
		time.Sleep(time.Duration(electionCooldownPeriod) * time.Second)
		b.Elect()
	}

}

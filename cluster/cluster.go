package cluster

import (
	"context"
	"fmt"
	"knh/bully"
	"knh/utils"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var log = utils.GetLogger()

type ClusterMonitor struct {
	k8  *kubernetes.Clientset
	ns  string
	svc string
}

type ClusterConfig struct {
	MasterURL      string
	KubeConfigPath string
}

// Returns a new Clustermonitor object

func NewClusterMonitor(clusterConfig *ClusterConfig) (*ClusterMonitor, error) {
	config, err := clientcmd.BuildConfigFromFlags(clusterConfig.MasterURL, clusterConfig.KubeConfigPath)
	if err != nil {
		log.Warn().
			Err(err).
			Str("id", "").
			Str("coordinator", "").
			Str("address", "").
			Msg("Failed to build flags for cluster config")
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Warn().
			Err(err).
			Str("id", "").
			Str("coordinator", "").
			Str("address", "").
			Msg("Failed to get kubernetes client")
		return nil, err
	}
	return &ClusterMonitor{k8: clientset, ns: os.Getenv("NAMESPACE"), svc: os.Getenv("SERVICE_NAME")}, nil

}

// Halt the program until the current pod ip appears in the endpoints
func (c *ClusterMonitor) blockTillEpRefresh(addr string) (map[string]string, error) {
	for {
		ep, err := c.k8.CoreV1().Endpoints(c.ns).Get(context.Background(), c.svc, metav1.GetOptions{})
		if err != nil {
			log.Warn().
				Err(err).
				Str("id", "").
				Str("coordinator", "").
				Str("address", "").
				Msg("Failed to get endpoints")
			return nil, err
		}
		eps := c.GetK8PeerMapFromEP(ep.Subsets)
		_, ok := eps[addr]
		if ok {
			log.Info().
				Str("id", "").
				Str("error", "").
				Str("coordinator", "").
				Str("address", "").
				Msg("Successfully refreshed endpoints")
			return eps, nil
		}
		log.Info().
			Str("id", "").
			Str("error", "").
			Str("coordinator", "").
			Str("address", "").
			Msg("Waiting 5s for endpoints to refresh")
		time.Sleep(time.Second * 5)
	}
}

// Get a map of all the peer ip's
func (c *ClusterMonitor) GetK8PeerMapFromEP(subsets []v1.EndpointSubset) map[string]string {
	peerMap := make(map[string]string)
	for _, subset := range subsets {
		for _, addr := range subset.Addresses {
			id := fmt.Sprintf("%s@%s", *addr.NodeName, strings.Replace(addr.IP, ".", "", -1))
			peerMap[id] = fmt.Sprintf("%s:%s", addr.IP, os.Getenv("CONTAINER_PORT"))
		}
	}
	return peerMap
}

func (c *ClusterMonitor) Monitor() error {
	addr := os.Getenv("POD_IP")
	port := os.Getenv("CONTAINER_PORT")
	if port == "" {
		port = "8080"
	}
	id := fmt.Sprintf("%s@%s", os.Getenv("NODE_NAME"), strings.Replace(addr, ".", "", -1))
	// wait till the endpoints are refreshed
	log.Info().
		Str("id", "").
		Str("error", "").
		Str("coordinator", "").
		Str("address", "").
		Msg("Started waiting for endpoints to refresh")
	epSubsets, err := c.blockTillEpRefresh(id)
	if err != nil {
		log.Warn().
			Err(err).
			Str("id", "").
			Str("coordinator", "").
			Str("address", "").
			Msg("Failed to get refreshed endpoints")
		return err
	}
	b, err := bully.NewBully(id, fmt.Sprintf("%s:%s", addr, port), epSubsets)
	if err != nil {
		log.Warn().
			Str("error", "").
			Str("id", "").
			Str("coordinator", "").
			Str("address", "").
			Msg("Failed to create bully")
		return err
	}
	epInformer := informers.NewSharedInformerFactory(c.k8, 0).Core().V1().Endpoints().Informer()
	epInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			ep, ok := newObj.(*v1.Endpoints)
			if ok && ep.Name == os.Getenv("SERVICE_NAME") && len(ep.Subsets) > 0 {
				log.Debug().
					Str("error", "").
					Str("id", b.ID).
					Str("coordinator", b.Coordinator()).
					Str("address", b.GetAddress()).
					Msg("Endpoints updated, updating peers")
				b.Connect(c.GetK8PeerMapFromEP(ep.Subsets))
			}
		},
	})
	stopper := make(chan struct{})
	defer close(stopper)
	go epInformer.Run(stopper)
	b.Run()
	return nil
}

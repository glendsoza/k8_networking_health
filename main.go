package main

import (
	"knh/cluster"
	"knh/utils"
)

var log = utils.GetLogger()

func main() {
	c, err := cluster.NewClusterMonitor(&cluster.ClusterConfig{})
	if err != nil {
		log.Fatal().
			Err(err).
			Str("id", "").
			Str("coordinator", "").
			Str("address", "").
			Msg("Failed to create cluster monitor")
	}
	err = c.Monitor()
	if err != nil {
		log.Fatal().
			Err(err).
			Str("id", "").
			Str("coordinator", "").
			Str("address", "").
			Msg("Failed to monitor the cluster")
	}
}

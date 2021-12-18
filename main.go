package main

import (
	"knh/cluster"
	"knh/utils"
	"os"
)

var log = utils.GetLogger()

func main() {
	for _, requiredEnv := range []string{"POD_IP", "NODE_NAME", "SERVICE_NAME", "NAMESPACE"} {
		if os.Getenv(requiredEnv) != "" {
			log.Fatal().
				Str("err", "").
				Str("id", "").
				Str("coordinator", "").
				Str("address", "").
				Msgf("Missiing required environment variable %s", requiredEnv)
		}
	}
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

# K8 Networking Health

KNH is a tool to check the networking health of your kubernetes cluster.

## Overview

KNH is a kubernetes networking health checker based on the bully alogrithm. It is meant to be deployed as a kubernetes daemonset or a deployment.
Once deployed through a leader election mechanism a coordinator is elected and the coordinator is responsible for communicating with other KNH pods in the cluster. In case coordinator fails to communicate with the pod a webhook is triggered to indicate the failure of the peer. In case of failure of coordinator through election mechanism a new coordinator is elected and the process will continue.

Status of all the nodes on which KNH pods are spawned will also be posted to a webhook

## How it works ?

KNH is based on the modified version of bully algorithm implimentation found here [TimTosi/bully-algorithm](https://github.com/TimTosi/bully-algorithm)

Each KNH pods hosts a bully which exposes two http endpoint `/ping` to respond to health check and `/coordinator` to set coordinator

Using the [client-go](https://github.com/kubernetes/client-go) package information about the IP's of other KNH pods are obtained and added as peers. IP's are updated after each endpoint refresh which may occur due to re-deployment or KNH pods being scaled up or down

In case coordinator fails to communicate with the pod a post request will me made to configured endpoint with the following payload

```
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "peer": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string"
        },
        "address": {
          "type": "string"
        },
        "alive": {
          "type": "boolean"
        },
        "node_name": {
          "type": "string"
        }
      },
      "required": [
        "id",
        "address",
        "alive",
        "node_name"
      ]
    },
    "coordinator": {
      "type": "string"
    },
    "coordinator_address": {
      "type": "string"
    }
  },
  "required": [
    "peer",
    "coordinator",
    "coordinator_address"
  ]
}
```

```
{"peer":{"id":"my-node1@103205","address":"10.32.0.5:8080","alive":false,"node_name":"my-node1"},"coordinator":"my-node@103209","coordinator_address":"10.32.0.9:8080"}
```

This webhook will be triggered once when the pod which was responsive becuase unresponsive, if the pod becomes responsive again it will be added back to list of peers with `alive` as true but no webhooks will be triggered

Satus of all the peers will also be posted after each leader election to configured endpoints with the following payload

```
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "object",
  "properties": {
    "peer_map": {
      "type": "array",
      "items": [
        {
          "type": "object",
          "properties": {
            "id": {
              "type": "string"
            },
            "address": {
              "type": "string"
            },
            "alive": {
              "type": "boolean"
            },
            "node_name": {
              "type": "string"
            }
          },
          "required": [
            "id",
            "address",
            "alive",
            "node_name"
          ]
        }
      ]
    },
    "coordinator": {
      "type": "string"
    },
    "coordinator_address": {
      "type": "string"
    }
  },
  "required": [
    "peer_map",
    "coordinator",
    "coordinator_address"
  ]
}
```

```
{"peer_map":[{"id":"my-node1@103205","address":"10.32.0.5:8080","alive":true,"node_name":"my-node1"}],"coordinator":"my-node0@103209","coordinator_address":"10.32.0.9:8080"}
```

## Installation and using it in the K8 Cluster

- Clone the repo
- Stat to the project root and build the docker image by running `docker build -t knh:v1.0 .`
- Create a sample deamonset as show here [daemonset.yaml](https://github.com/glendsoza/k8_networking_health/example/daemonset.yaml)
- Run kubectl apply -f `daemonset.yaml`
- Verify all the pods are up and running

## Configuration

Time units are in seconds 

| Name                     	| Required 	| Default 	| Description                                                                                                                                                                                                                                                                  	|
|--------------------------	|----------	|---------	|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------	|
| POD_IP                   	| Yes      	| -       	| IP of the pod that has to be passed to container as environment variable                                                                                                                                                                                                     	|
| MY_NODE_NAME             	| Yes      	| -       	| Node Name on which the pod has spawned, this is used to generate id for a the bully and while making post request to cluster health end point url's                                                                                                                          	|
| SERVICE_NAME             	| Yes      	| -       	| Service name targeting the Daemonset/Deployment                                                                                                                                                                                                                              	|
| NAMESPACE                	| Yes      	| -       	| Namespace of the Daemonset/Deployment                                                                                                                                                                                                                                        	|
| CONNECT_MAX_RETRIES      	| No       	| 5       	| Each pod ties to establish connection to peer, this environment variable controls how many times pod has to retry before giving up on establishing connection                                                                                                                	|
| CONNECT_COOLDOWN_PERIOD  	| No       	| 2       	| Time between each connection retries attempts                                                                                                                                                                                                                                	|
| SEND_MAX_RETRIES         	| No       	| 5       	| Coordinator sends election to other pods, this this environment variable controls how many times pod has to retry sending election to other pod before giving up in case it fails to send/<br>Note : Pods to which coordinator fails to send election will be marked as dead 	|
| SEND_COOLDOWN_PERIOD     	| No       	| 1       	| Time between each send retry attempts                                                                                                                                                                                                                                        	|
| ELECTION_COOLDOWN_PERIOD 	| No       	| 15      	| Time to sleep after each election                                                                                                                                                                                                                                            	|
| SANITY_CHECK_URL         	| No       	| -       	| Coordinator will query the given url to determine if its healthy or not, this is done to ensure that in case of coordinator failure multiple post request wont be send cluster health endpoints                                                                              	|
| PEER_STATUS_URL          	| No       	| -       	| URL to send the dead peer status as post request                                                                                                                                                                                                                             	|
| CLUSTER_STATUS_URL       	| No       	| -       	| URL to send the cluster status as post request                                                                                                                                                                                                                               	|
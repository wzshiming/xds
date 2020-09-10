package utils

import (
	"fmt"

	structpb "github.com/golang/protobuf/ptypes/struct"
)

// NodeConfig for the Client connection.
type NodeConfig struct {
	NodeID string

	// Namespace defaults to 'default'
	Namespace string

	// Workload defaults to 'test'
	Workload string

	// NodeType defaults to 'sidecar'. 'ingress' and 'router' are also supported.
	NodeType string

	// IP is currently the primary key used to locate inbound configs. It is sent by client,
	// must match a known endpoint IP. Tests can use a ServiceEntry to register fake IPs.
	IP string

	// Cluster defaults to 'svc.cluster.local'
	Cluster string

	// Metadata includes additional metadata for the node
	Metadata map[string]interface{}
}

func (c *NodeConfig) ID() string {
	if c.NodeID == "" {
		if c.Namespace == "" {
			c.Namespace = "default"
		}
		if c.NodeType == "" {
			c.NodeType = "sidecar"
		}
		if c.IP == "" {
			c.IP = GetPrivateIPIfAvailable().String()
		}
		if c.Workload == "" {
			c.Workload = "test"
		}
		if c.Cluster == "" {
			c.Cluster = "svc.cluster.local"
		}

		c.NodeID = fmt.Sprintf("%s~%s~%s.%s~%s.%s", c.NodeType, c.IP, c.Workload, c.Namespace, c.Namespace, c.Cluster)
	}
	return c.NodeID
}

func (c *NodeConfig) Meta() *structpb.Struct {
	return MustMapToProtoStruct(c.Metadata)
}

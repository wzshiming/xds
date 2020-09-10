package xds_v2

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/wzshiming/xds/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	ClusterType  = resource.ClusterType
	EndpointType = resource.EndpointType
	ListenerType = resource.ListenerType
	RouteType    = resource.RouteType
	SecretType   = resource.SecretType
	RuntimeType  = resource.RuntimeType
	AnyType      = resource.AnyType
)

// Config for the Client connection.
type Config struct {
	utils.NodeConfig
	OnConnect      func(cli *Client) error
	ContextDialer  func(ctx context.Context, address string) (net.Conn, error)
	HandleCDS      func(cli *Client, clusters []*envoy_api_v2.Cluster)
	HandleEDS      func(cli *Client, endpoints []*envoy_api_v2.ClusterLoadAssignment)
	HandleLDS      func(cli *Client, listeners []*envoy_api_v2.Listener)
	HandleRDS      func(cli *Client, routes []*envoy_api_v2.RouteConfiguration)
	HandleSDS      func(cli *Client, secrets []*envoy_api_v2_auth.Secret)
	HandleNotFound func(cli *Client, others []*any.Any)
}

// Client implements a client for ADS.
type Client struct {
	stream    envoy_service_discovery_v2.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	conn      *grpc.ClientConn
	tlsConfig *tls.Config
	url       string
	isClose   bool
	node      *envoy_api_v2_core.Node
	Config
}

// NewClient connects to a Client server, with optional TLS authentication if a cert dir is specified.
func NewClient(url string, tlsConfig *tls.Config, opts *Config) *Client {
	ads := &Client{
		tlsConfig: tlsConfig,
		url:       url,
	}
	if opts != nil {
		ads.Config = *opts
	}
	return ads
}

// Clone the once.
func (c *Client) Clone() *Client {
	return NewClient(c.url, c.tlsConfig.Clone(), &c.Config)
}

// Close the once.
func (c *Client) Close() error {
	c.isClose = true
	if c.stream != nil {
		c.stream.CloseSend()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Run the ADS client.
func (c *Client) Run(ctx context.Context) error {
	err := c.run(ctx)
	if err != nil {
		return err
	}
	return c.handleRecv()
}

func (c *Client) Start(ctx context.Context) error {
	err := c.run(ctx)
	if err != nil {
		return err
	}
	go c.handleRecv()
	return nil
}

func (c *Client) run(ctx context.Context) error {
	opts := []grpc.DialOption{}
	if c.tlsConfig != nil {
		secret := credentials.NewTLS(c.tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(secret))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	if c.ContextDialer != nil {
		opts = append(opts, grpc.WithContextDialer(c.ContextDialer))
	}
	conn, err := grpc.DialContext(ctx, c.url, opts...)
	if err != nil {
		return err
	}

	xds := envoy_service_discovery_v2.NewAggregatedDiscoveryServiceClient(conn)

	stm, err := xds.StreamAggregatedResources(ctx)
	if err != nil {
		return err
	}

	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = conn
	if c.stream != nil {
		c.stream.CloseSend()
	}
	c.stream = stm
	if c.OnConnect != nil {
		return c.OnConnect(c)
	}
	return nil
}

func (c *Client) handleRecv() error {
	c.isClose = false
	clusters := []*envoy_api_v2.Cluster{}
	endpoints := []*envoy_api_v2.ClusterLoadAssignment{}
	listeners := []*envoy_api_v2.Listener{}
	routes := []*envoy_api_v2.RouteConfiguration{}
	secrets := []*envoy_api_v2_auth.Secret{}
	others := []*any.Any{}
	ctx := c.stream.Context()
	for {
		err := ctx.Err()
		if err != nil {
			return err
		}
		msg, err := c.stream.Recv()
		if err != nil {
			if code := status.Code(err); code == codes.Canceled || code == codes.DeadlineExceeded {
				return nil
			}
			return fmt.Errorf("connection closed : error: %w", err)
		}

		clusters = clusters[:0]
		endpoints = endpoints[:0]
		listeners = listeners[:0]
		routes = routes[:0]
		secrets = secrets[:0]
		others = others[:0]

		for _, rsc := range msg.Resources {
			switch rsc.TypeUrl {
			case ClusterType:
				ll := &envoy_api_v2.Cluster{}
				_ = proto.Unmarshal(rsc.Value, ll)
				clusters = append(clusters, ll)
			case EndpointType:
				ll := &envoy_api_v2.ClusterLoadAssignment{}
				_ = proto.Unmarshal(rsc.Value, ll)
				endpoints = append(endpoints, ll)
			case ListenerType:
				ll := &envoy_api_v2.Listener{}
				_ = proto.Unmarshal(rsc.Value, ll)
				listeners = append(listeners, ll)
			case RouteType:
				ll := &envoy_api_v2.RouteConfiguration{}
				_ = proto.Unmarshal(rsc.Value, ll)
				routes = append(routes, ll)
			case SecretType:
				ll := &envoy_api_v2_auth.Secret{}
				_ = proto.Unmarshal(rsc.Value, ll)
				secrets = append(secrets, ll)
			default:
				others = append(others, rsc)
			}
		}

		if len(clusters) != 0 && c.HandleCDS != nil {
			c.HandleCDS(c, clusters)
		}
		if len(endpoints) != 0 && c.HandleEDS != nil {
			c.HandleEDS(c, endpoints)
		}
		if len(listeners) != 0 && c.HandleLDS != nil {
			c.HandleLDS(c, listeners)
		}
		if len(routes) != 0 && c.HandleRDS != nil {
			c.HandleRDS(c, routes)
		}
		if len(secrets) != 0 && c.HandleSDS != nil {
			c.HandleSDS(c, secrets)
		}
		if len(others) != 0 && c.HandleNotFound != nil {
			c.HandleNotFound(c, others)
		}

		c.ack(msg)
	}
}

func (c *Client) Node() *envoy_api_v2_core.Node {
	if c.node == nil {
		c.node = &envoy_api_v2_core.Node{
			Id:       c.NodeConfig.ID(),
			Metadata: c.NodeConfig.Meta(),
		}
	}
	return c.node
}

func (c *Client) Send(req *envoy_api_v2.DiscoveryRequest) error {
	req.Node = c.Node()
	return c.stream.Send(req)
}

func (c *Client) SendRsc(typeURL string, rsc []string) error {
	return c.Send(&envoy_api_v2.DiscoveryRequest{
		ResponseNonce: "",
		TypeUrl:       typeURL,
		ResourceNames: rsc,
	})
}

func (c *Client) ack(msg *envoy_api_v2.DiscoveryResponse) error {
	return c.Send(&envoy_api_v2.DiscoveryRequest{
		ResponseNonce: msg.Nonce,
		TypeUrl:       msg.TypeUrl,
		VersionInfo:   msg.VersionInfo,
	})
}

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/wzshiming/xds/utils"
	xds_v2 "github.com/wzshiming/xds/v2"
	xds_v3 "github.com/wzshiming/xds/v3"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var (
	url      = "127.0.0.1:15010"
	certs    = ""
	nodeId   = ""
	ver      = uint64(2)
	metadata = map[string]interface{}{}
)

func init() {
	flag.StringVar(&url, "u", url, "xds server")
	flag.StringVar(&certs, "c", certs, "certs folder {cert-chain.pem,key.pem,root-cert.pem}")
	flag.StringVar(&nodeId, "n", nodeId, "node id")
	flag.Uint64Var(&ver, "v", ver, "xds version (2/3)")
	metadataJSON := "{}"
	flag.StringVar(&metadataJSON, "m", metadataJSON, "node metadata")
	flag.Parse()

	err := json.Unmarshal([]byte(metadataJSON), &metadata)
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	ctx := context.Background()
	switch ver {
	case 2:
		mainV2(ctx)
	case 3:
		mainV3(ctx)
	default:
		flag.PrintDefaults()
	}
}

func mainV2(ctx context.Context) {
	var tlsConfig *tls.Config
	if certs != "" {
		t, err := utils.TlsConfigFromDir(certs)
		if err != nil {
			log.Fatalln(err)
		}
		tlsConfig = t
	}

	conf := xds_v2.Config{}

	send := func(cli *xds_v2.Client, typeurl string, rsc []string) {
		err := cli.SendRsc(typeurl, rsc)
		if err != nil {
			log.Println(err)
		}
	}
	conf.HandleCDS = func(cli *xds_v2.Client, clusters []*envoy_api_v2.Cluster) {
		log.Println("Response CDS", len(clusters))
		sort.Slice(clusters, func(i, j int) bool {
			return clusters[i].Name < clusters[j].Name
		})
		names := []string{}
		for _, cluster := range clusters {
			show(cluster)
			names = append(names, xds_v2.GetEndpointNames(cluster)...)
		}
		log.Println("Request EDS", len(names), strings.Join(names, ","))
		send(cli, xds_v2.EndpointType, names)
	}
	conf.HandleLDS = func(cli *xds_v2.Client, listeners []*envoy_api_v2.Listener) {
		log.Println("Response LDS", len(listeners))
		sort.Slice(listeners, func(i, j int) bool {
			return listeners[i].Name < listeners[j].Name
		})
		names := []string{}
		for _, listener := range listeners {
			show(listener)
			names = append(names, xds_v2.GetRouteNames(listener)...)
		}
		log.Println("Request RDS", len(names), strings.Join(names, ","))
		send(cli, xds_v2.RouteType, names)
	}
	conf.HandleRDS = func(cli *xds_v2.Client, routes []*envoy_api_v2.RouteConfiguration) {
		log.Println("Response RDS", len(routes))
		sort.Slice(routes, func(i, j int) bool {
			return routes[i].Name < routes[j].Name
		})
		for _, route := range routes {
			show(route)
		}
	}
	conf.HandleEDS = func(cli *xds_v2.Client, endpoints []*envoy_api_v2.ClusterLoadAssignment) {
		log.Println("Response EDS", len(endpoints))
		sort.Slice(endpoints, func(i, j int) bool {
			return endpoints[i].ClusterName < endpoints[j].ClusterName
		})
		for _, endpoint := range endpoints {
			show(endpoint)
		}
	}
	conf.HandleSDS = func(cli *xds_v2.Client, secrets []*envoy_api_v2_auth.Secret) {
		log.Println("Response SDS", len(secrets))
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].Name < secrets[j].Name
		})
		for _, secret := range secrets {
			show(secret)
		}
	}
	conf.OnConnect = func(cli *xds_v2.Client) error {
		log.Println("Request CDS", 0)
		send(cli, xds_v2.ClusterType, nil)
		log.Println("Request LDS", 0)
		send(cli, xds_v2.ListenerType, nil)
		return nil
	}
	conf.NodeConfig.NodeID = nodeId
	conf.NodeConfig.Metadata = metadata

	cli := xds_v2.NewClient(url, tlsConfig, &conf)
	err := cli.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}

func mainV3(ctx context.Context) {
	var tlsConfig *tls.Config
	if certs != "" {
		t, err := utils.TlsConfigFromDir(certs)
		if err != nil {
			log.Fatalln(err)
		}
		tlsConfig = t
	}

	conf := xds_v3.Config{}

	send := func(cli *xds_v3.Client, typeurl string, rsc []string) {
		err := cli.SendRsc(typeurl, rsc)
		if err != nil {
			log.Println(err)
		}
	}
	conf.HandleCDS = func(cli *xds_v3.Client, clusters []*envoy_config_cluster_v3.Cluster) {
		log.Println("Response CDS", len(clusters))
		sort.Slice(clusters, func(i, j int) bool {
			return clusters[i].Name < clusters[j].Name
		})
		names := []string{}
		for _, cluster := range clusters {
			show(cluster)
			names = append(names, xds_v3.GetEndpointNames(cluster)...)
		}
		log.Println("Request EDS", len(names), strings.Join(names, ","))
		send(cli, xds_v3.EndpointType, names)
	}
	conf.HandleLDS = func(cli *xds_v3.Client, listeners []*envoy_config_listener_v3.Listener) {
		log.Println("Response LDS", len(listeners))
		sort.Slice(listeners, func(i, j int) bool {
			return listeners[i].Name < listeners[j].Name
		})
		names := []string{}
		for _, listener := range listeners {
			show(listener)
			names = append(names, xds_v3.GetRouteNames(listener)...)
		}
		log.Println("Request RDS", len(names), strings.Join(names, ","))
		send(cli, xds_v3.RouteType, names)
	}
	conf.HandleRDS = func(cli *xds_v3.Client, routes []*envoy_config_route_v3.RouteConfiguration) {
		log.Println("Response RDS", len(routes))
		sort.Slice(routes, func(i, j int) bool {
			return routes[i].Name < routes[j].Name
		})
		for _, route := range routes {
			show(route)
		}
	}
	conf.HandleEDS = func(cli *xds_v3.Client, endpoints []*envoy_config_endpoint_v3.ClusterLoadAssignment) {
		log.Println("Response EDS", len(endpoints))
		sort.Slice(endpoints, func(i, j int) bool {
			return endpoints[i].ClusterName < endpoints[j].ClusterName
		})
		for _, endpoint := range endpoints {
			show(endpoint)
		}
	}
	conf.HandleSDS = func(cli *xds_v3.Client, secrets []*envoy_extensions_transport_sockets_tls_v3.Secret) {
		log.Println("Response SDS", len(secrets))
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].Name < secrets[j].Name
		})
		for _, secret := range secrets {
			show(secret)
		}
	}
	conf.OnConnect = func(cli *xds_v3.Client) error {
		log.Println("Request CDS", 0)
		send(cli, xds_v3.ClusterType, nil)
		log.Println("Request LDS", 0)
		send(cli, xds_v3.ListenerType, nil)
		return nil
	}
	conf.NodeConfig.NodeID = nodeId
	conf.NodeConfig.Metadata = metadata

	cli := xds_v3.NewClient(url, tlsConfig, &conf)
	err := cli.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}

var jsonpbMarshaler = jsonpb.Marshaler{
	AnyResolver: dynamicAnyResolver{},
}

func show(m proto.Message) {
	data, err := jsonpbMarshaler.MarshalToString(m)
	if err != nil {
		log.Println(err)
	}
	fmt.Println(data)
}

type dynamicAnyResolver struct {
}

func (dynamicAnyResolver) Resolve(typeURL string) (proto.Message, error) {
	mt, err := protoregistry.GlobalTypes.FindMessageByURL(typeURL)
	if err != nil {
		log.Println(err, typeURL)
		return &empty.Empty{}, nil
	}
	return proto.MessageV1(mt.New().Interface()), nil
}

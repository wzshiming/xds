package xds_v3

import (
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

// GetEndpointNames returns the EDS names for CDS
func GetEndpointNames(v *envoy_config_cluster_v3.Cluster) []string {
	names := []string{}
	switch typ := v.ClusterDiscoveryType.(type) {
	case *envoy_config_cluster_v3.Cluster_Type:
		if typ.Type == envoy_config_cluster_v3.Cluster_EDS {
			if v.EdsClusterConfig != nil && v.EdsClusterConfig.ServiceName != "" {
				names = append(names, v.EdsClusterConfig.ServiceName)
			} else {
				names = append(names, v.Name)
			}
		}
	}
	return names
}

// GetRouteNames returns the RDS names for LDS
func GetRouteNames(v *envoy_config_listener_v3.Listener) []string {
	names := []string{}
	for _, chain := range v.FilterChains {
		for _, filter := range chain.Filters {
			if filter.Name != wellknown.HTTPConnectionManager {
				continue
			}
			config := resource.GetHTTPConnectionManager(filter)
			if config == nil {
				continue
			}
			if rds, ok := config.RouteSpecifier.(*envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager_Rds); ok && rds != nil && rds.Rds != nil {
				names = append(names, rds.Rds.RouteConfigName)
			}
		}
	}
	return names
}

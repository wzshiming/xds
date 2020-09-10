package xds_v2

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_config_filter_network_http_connection_manager_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

// GetEndpointNames returns the EDS names for CDS
func GetEndpointNames(v *envoy_api_v2.Cluster) []string {
	names := []string{}
	switch typ := v.ClusterDiscoveryType.(type) {
	case *envoy_api_v2.Cluster_Type:
		if typ.Type == envoy_api_v2.Cluster_EDS {
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
func GetRouteNames(v *envoy_api_v2.Listener) []string {
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
			if rds, ok := config.RouteSpecifier.(*envoy_config_filter_network_http_connection_manager_v2.HttpConnectionManager_Rds); ok && rds != nil && rds.Rds != nil {
				names = append(names, rds.Rds.RouteConfigName)
			}
		}
	}
	return names
}

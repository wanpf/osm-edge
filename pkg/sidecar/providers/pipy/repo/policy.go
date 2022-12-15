package repo

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	multiclusterv1alpha1 "github.com/openservicemesh/osm/pkg/apis/multicluster/v1alpha1"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/sidecar/providers/pipy/registry"
	"github.com/openservicemesh/osm/pkg/utils/cidr"
)

var (
	addrWithPort, _ = regexp.Compile(`:\d+$`)
)

func (p *PipyConf) setSidecarLogLevel(sidecarLogLevel string) (update bool) {
	if update = !strings.EqualFold(p.Spec.SidecarLogLevel, sidecarLogLevel); update {
		p.Spec.SidecarLogLevel = sidecarLogLevel
	}
	return
}

func (p *PipyConf) setLocalDNSProxy(enable bool, primary, secondary string) {
	if enable {
		p.Spec.LocalDNSProxy = new(LocalDNSProxy)
		if len(primary) > 0 || len(secondary) > 0 {
			p.Spec.LocalDNSProxy.UpstreamDNSServers = new(UpstreamDNSServers)
			if len(primary) > 0 {
				p.Spec.LocalDNSProxy.UpstreamDNSServers.Primary = &primary
			}
			if len(secondary) > 0 {
				p.Spec.LocalDNSProxy.UpstreamDNSServers.Secondary = &secondary
			}
		}
	} else {
		p.Spec.LocalDNSProxy = nil
	}
}

func (p *PipyConf) setEnableSidecarActiveHealthChecks(enableSidecarActiveHealthChecks bool) (update bool) {
	if update = p.Spec.FeatureFlags.EnableSidecarActiveHealthChecks != enableSidecarActiveHealthChecks; update {
		p.Spec.FeatureFlags.EnableSidecarActiveHealthChecks = enableSidecarActiveHealthChecks
	}
	return
}

func (p *PipyConf) setEnableEgress(enableEgress bool) (update bool) {
	if update = p.Spec.Traffic.EnableEgress != enableEgress; update {
		p.Spec.Traffic.EnableEgress = enableEgress
	}
	return
}

func (p *PipyConf) setEnablePermissiveTrafficPolicyMode(enablePermissiveTrafficPolicyMode bool) (update bool) {
	if update = p.Spec.Traffic.enablePermissiveTrafficPolicyMode != enablePermissiveTrafficPolicyMode; update {
		p.Spec.Traffic.enablePermissiveTrafficPolicyMode = enablePermissiveTrafficPolicyMode
	}
	return
}

func (p *PipyConf) isPermissiveTrafficPolicyMode() bool {
	return p.Spec.Traffic.enablePermissiveTrafficPolicyMode
}

func (p *PipyConf) newInboundTrafficPolicy() *InboundTrafficPolicy {
	if p.Inbound == nil {
		p.Inbound = new(InboundTrafficPolicy)
	}
	return p.Inbound
}

func (p *PipyConf) newOutboundTrafficPolicy() *OutboundTrafficPolicy {
	if p.Outbound == nil {
		p.Outbound = new(OutboundTrafficPolicy)
	}
	return p.Outbound
}

func (p *PipyConf) newForwardTrafficPolicy() *ForwardTrafficPolicy {
	if p.Forward == nil {
		p.Forward = new(ForwardTrafficPolicy)
	}
	return p.Forward
}

func (p *PipyConf) rebalancedOutboundClusters() {
	if p.Outbound == nil {
		return
	}
	if p.Outbound.ClustersConfigs == nil || len(p.Outbound.ClustersConfigs) == 0 {
		return
	}
	for _, clusterConfigs := range p.Outbound.ClustersConfigs {
		weightedEndpoints := clusterConfigs.Endpoints
		if weightedEndpoints == nil || len(*weightedEndpoints) == 0 {
			continue
		}
		hasLocalEndpoints := false
		for _, wze := range *weightedEndpoints {
			if len(wze.Cluster) == 0 {
				hasLocalEndpoints = true
				break
			}
		}
		for _, wze := range *weightedEndpoints {
			if len(wze.Cluster) > 0 {
				if multiclusterv1alpha1.FailOverLbType == multiclusterv1alpha1.LoadBalancerType(wze.LBType) {
					if hasLocalEndpoints {
						wze.Weight = constants.ClusterWeightFailOver
					} else {
						wze.Weight = constants.ClusterWeightAcceptAll
					}
				} else if multiclusterv1alpha1.ActiveActiveLbType == multiclusterv1alpha1.LoadBalancerType(wze.LBType) {
					if wze.Weight == 0 {
						wze.Weight = constants.ClusterWeightAcceptAll
					}
				}
			} else {
				if wze.Weight == 0 {
					wze.Weight = constants.ClusterWeightAcceptAll
				}
			}
		}
	}
}

func (p *PipyConf) rebalancedForwardClusters() {
	if p.Forward == nil {
		return
	}
	if p.Forward.ForwardMatches != nil && len(p.Forward.ForwardMatches) > 0 {
		for _, weightedEndpoints := range p.Forward.ForwardMatches {
			if len(weightedEndpoints) == 0 {
				continue
			}
			for upstreamEndpoint, weight := range weightedEndpoints {
				if weight == 0 {
					(weightedEndpoints)[upstreamEndpoint] = constants.ClusterWeightAcceptAll
				}
			}
		}
	}
	if p.Forward.EgressGateways != nil && len(p.Forward.EgressGateways) > 0 {
		for _, clusterConfigs := range p.Forward.EgressGateways {
			weightedEndpoints := clusterConfigs.Endpoints
			if weightedEndpoints == nil || len(*weightedEndpoints) == 0 {
				continue
			}
			missingWeightNb := 0
			availableWeight := uint32(100)
			for _, wze := range *weightedEndpoints {
				if wze.Weight == 0 {
					missingWeightNb++
				} else {
					availableWeight = availableWeight - uint32(wze.Weight)
				}
			}

			if missingWeightNb == len(*weightedEndpoints) {
				for _, wze := range *weightedEndpoints {
					if wze.Weight == 0 {
						wze.Weight = Weight(availableWeight / uint32(missingWeightNb))
						missingWeightNb--
						availableWeight = availableWeight - uint32(wze.Weight)
					}
				}
			}
		}
	}
}

func (p *PipyConf) copyAllowedEndpoints(kubeController k8s.Controller, proxyRegistry *registry.ProxyRegistry) bool {
	ready := true
	p.AllowedEndpoints = make(map[string]string)
	allPods := kubeController.ListPods()
	for _, pod := range allPods {
		proxyUUID, err := GetProxyUUIDFromPod(pod)
		if err != nil {
			continue
		}
		proxy := proxyRegistry.GetConnectedProxy(proxyUUID)
		if proxy == nil {
			ready = false
			continue
		}
		p.AllowedEndpoints[proxy.GetAddr()] = fmt.Sprintf("%s.%s", pod.Namespace, pod.Name)
		if len(proxy.GetAddr()) == 0 {
			ready = false
		}
	}
	if p.Inbound == nil {
		return ready
	}
	if len(p.Inbound.TrafficMatches) == 0 {
		return ready
	}
	for _, trafficMatch := range p.Inbound.TrafficMatches {
		if len(trafficMatch.SourceIPRanges) == 0 {
			continue
		}
		for ipRange := range trafficMatch.SourceIPRanges {
			ingressIP := strings.TrimSuffix(string(ipRange), "/32")
			p.AllowedEndpoints[ingressIP] = "Ingress Controller"
		}
	}
	return ready
}

func (itm *InboundTrafficMatch) addSourceIPRange(ipRange SourceIPRange, sourceSpec *SourceSecuritySpec) {
	if itm.SourceIPRanges == nil {
		itm.SourceIPRanges = make(map[SourceIPRange]*SourceSecuritySpec)
	}
	if _, exists := itm.SourceIPRanges[ipRange]; !exists {
		itm.SourceIPRanges[ipRange] = sourceSpec
	}
}

func (itm *InboundTrafficMatch) addAllowedEndpoint(address Address, serviceName ServiceName) {
	if itm.AllowedEndpoints == nil {
		itm.AllowedEndpoints = make(AllowedEndpoints)
	}
	if _, exists := itm.AllowedEndpoints[address]; !exists {
		itm.AllowedEndpoints[address] = serviceName
	}
}

func (itm *InboundTrafficMatch) setTCPServiceRateLimit(rateLimit *policyv1alpha1.RateLimitSpec) {
	if rateLimit == nil || rateLimit.Local == nil {
		itm.RateLimit = nil
	} else {
		itm.RateLimit = newTCPRateLimit(rateLimit.Local)
	}
}

func (otm *OutboundTrafficMatch) addDestinationIPRange(ipRange DestinationIPRange, destinationSpec *DestinationSecuritySpec) {
	if otm.DestinationIPRanges == nil {
		otm.DestinationIPRanges = make(map[DestinationIPRange]*DestinationSecuritySpec)
	}
	if _, exists := otm.DestinationIPRanges[ipRange]; !exists {
		otm.DestinationIPRanges[ipRange] = destinationSpec
	}
}

func (otm *OutboundTrafficMatch) setServiceIdentity(serviceIdentity identity.ServiceIdentity) {
	otm.ServiceIdentity = serviceIdentity
}

func (otm *OutboundTrafficMatch) setAllowedEgressTraffic(allowedEgressTraffic bool) {
	otm.AllowedEgressTraffic = allowedEgressTraffic
}

func (itm *InboundTrafficMatch) setPort(port Port) {
	itm.Port = port
}

func (otm *OutboundTrafficMatch) setPort(port Port) {
	otm.Port = port
}

func (otm *OutboundTrafficMatch) setEgressForwardGateway(egresssGateway *string) {
	otm.EgressForwardGateway = egresssGateway
}

func (itm *InboundTrafficMatch) setProtocol(protocol Protocol) {
	protocol = Protocol(strings.ToLower(string(protocol)))
	if constants.ProtocolTCPServerFirst == protocol {
		itm.Protocol = constants.ProtocolTCP
	} else {
		itm.Protocol = protocol
	}
}

func (otm *OutboundTrafficMatch) setProtocol(protocol Protocol) {
	protocol = Protocol(strings.ToLower(string(protocol)))
	if constants.ProtocolTCPServerFirst == protocol {
		otm.Protocol = constants.ProtocolTCP
	} else {
		otm.Protocol = protocol
	}
}

func (itm *InboundTrafficMatch) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if itm.TargetClusters == nil {
		itm.TargetClusters = make(WeightedClusters)
	}
	itm.TargetClusters[clusterName] = weight
}

func (otm *OutboundTrafficMatch) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if otm.TargetClusters == nil {
		otm.TargetClusters = make(WeightedClusters)
	}
	otm.TargetClusters[clusterName] = weight
}

func (itm *InboundTrafficMatch) addHTTPHostPort2Service(hostPort HTTPHostPort, ruleName HTTPRouteRuleName) {
	if itm.HTTPHostPort2Service == nil {
		itm.HTTPHostPort2Service = make(HTTPHostPort2Service)
	}
	itm.HTTPHostPort2Service[hostPort] = ruleName
}

func (otm *OutboundTrafficMatch) addHTTPHostPort2Service(hostPort HTTPHostPort, ruleName HTTPRouteRuleName) {
	if otm.HTTPHostPort2Service == nil {
		otm.HTTPHostPort2Service = make(HTTPHostPort2Service)
	}
	otm.HTTPHostPort2Service[hostPort] = ruleName
}

func (itm *InboundTrafficMatch) newHTTPServiceRouteRules(httpRouteRuleName HTTPRouteRuleName) *InboundHTTPRouteRules {
	if itm.HTTPServiceRouteRules == nil {
		itm.HTTPServiceRouteRules = make(InboundHTTPServiceRouteRules)
	}
	if len(httpRouteRuleName) == 0 {
		return nil
	}
	rules, exist := itm.HTTPServiceRouteRules[httpRouteRuleName]
	if !exist || rules == nil {
		newCluster := new(InboundHTTPRouteRules)
		itm.HTTPServiceRouteRules[httpRouteRuleName] = newCluster
		return newCluster
	}
	return rules
}

func (otm *OutboundTrafficMatch) newHTTPServiceRouteRules(httpRouteRuleName HTTPRouteRuleName) *OutboundHTTPRouteRules {
	if otm.HTTPServiceRouteRules == nil {
		otm.HTTPServiceRouteRules = make(OutboundHTTPServiceRouteRules)
	}
	if len(httpRouteRuleName) == 0 {
		return nil
	}
	rules, exist := otm.HTTPServiceRouteRules[httpRouteRuleName]
	if !exist || rules == nil {
		newCluster := new(OutboundHTTPRouteRules)
		otm.HTTPServiceRouteRules[httpRouteRuleName] = newCluster
		return newCluster
	}
	return rules
}

func (itp *InboundTrafficPolicy) newTrafficMatch(port Port) *InboundTrafficMatch {
	if itp.TrafficMatches == nil {
		itp.TrafficMatches = make(InboundTrafficMatches)
	}
	trafficMatch, exist := itp.TrafficMatches[port]
	if !exist || trafficMatch == nil {
		trafficMatch = new(InboundTrafficMatch)
		itp.TrafficMatches[port] = trafficMatch
		return trafficMatch
	}
	return trafficMatch
}

func (itp *InboundTrafficPolicy) getTrafficMatch(port Port) *InboundTrafficMatch {
	if itp.TrafficMatches == nil {
		return nil
	}
	if trafficMatch, exist := itp.TrafficMatches[port]; exist {
		return trafficMatch
	}
	return nil
}

func (otp *OutboundTrafficPolicy) newTrafficMatch(port Port, name string) (*OutboundTrafficMatch, bool) {
	namedPort := fmt.Sprintf(`%d=%s`, port, name)
	if otp.namedTrafficMatches == nil {
		otp.namedTrafficMatches = make(namedOutboundTrafficMatches)
	}
	trafficMatch, exists := otp.namedTrafficMatches[namedPort]
	if exists {
		return trafficMatch, true
	}

	trafficMatch = new(OutboundTrafficMatch)
	otp.namedTrafficMatches[namedPort] = trafficMatch

	if otp.TrafficMatches == nil {
		otp.TrafficMatches = make(OutboundTrafficMatches)
	}
	trafficMatches := otp.TrafficMatches[port]
	trafficMatches = append(trafficMatches, trafficMatch)
	otp.TrafficMatches[port] = trafficMatches
	return trafficMatch, false
}

func (hrrs *InboundHTTPRouteRules) setHTTPServiceRateLimit(rateLimit *policyv1alpha1.RateLimitSpec) {
	if rateLimit == nil || rateLimit.Local == nil {
		hrrs.RateLimit = nil
	} else {
		hrrs.RateLimit = newHTTPRateLimit(rateLimit.Local)
	}
}

func (hrrs *InboundHTTPRouteRules) setHTTPHeadersRateLimit(rateLimit *[]policyv1alpha1.HTTPHeaderSpec) {
	if rateLimit == nil {
		hrrs.HeaderRateLimits = nil
	} else {
		hrrs.HeaderRateLimits = newHTTPHeaderRateLimit(rateLimit)
	}
}

func (hrrs *InboundHTTPRouteRules) newHTTPServiceRouteRule(matchRule *HTTPMatchRule) (route *InboundHTTPRouteRule, duplicate bool) {
	for _, routeRule := range hrrs.RouteRules {
		if reflect.DeepEqual(*matchRule, routeRule.HTTPMatchRule) {
			return routeRule, true
		}
	}

	routeRule := new(InboundHTTPRouteRule)
	routeRule.HTTPMatchRule = *matchRule
	hrrs.RouteRules = append(hrrs.RouteRules, routeRule)
	return routeRule, false
}

func (hrrs *OutboundHTTPRouteRules) newHTTPServiceRouteRule(matchRule *HTTPMatchRule) (route *OutboundHTTPRouteRule, duplicate bool) {
	for _, routeRule := range hrrs.RouteRules {
		if reflect.DeepEqual(*matchRule, routeRule.HTTPMatchRule) {
			return routeRule, true
		}
	}

	routeRule := new(OutboundHTTPRouteRule)
	routeRule.HTTPMatchRule = *matchRule
	hrrs.RouteRules = append(hrrs.RouteRules, routeRule)
	return routeRule, false
}

func (hmr *HTTPMatchRule) addHeaderMatch(header Header, headerRegexp HeaderRegexp) {
	if hmr.Headers == nil {
		hmr.Headers = make(Headers)
	}
	hmr.Headers[header] = headerRegexp
}

func (hmr *HTTPMatchRule) addMethodMatch(method Method) {
	if hmr.allowedAnyMethod {
		return
	}
	if "*" == method {
		hmr.allowedAnyMethod = true
	}
	if hmr.allowedAnyMethod {
		hmr.Methods = nil
	} else {
		hmr.Methods = append(hmr.Methods, method)
	}
}

func (hrr *HTTPRouteRule) addWeightedCluster(clusterName ClusterName, weight Weight) {
	if hrr.TargetClusters == nil {
		hrr.TargetClusters = make(WeightedClusters)
	}
	hrr.TargetClusters[clusterName] = weight
}

func (hrr *HTTPRouteRule) addAllowedService(serviceName ServiceName) {
	if hrr.allowedAnyService {
		return
	}
	if "*" == serviceName {
		hrr.allowedAnyService = true
	}
	if hrr.allowedAnyService {
		hrr.AllowedServices = nil
	} else {
		hrr.AllowedServices = append(hrr.AllowedServices, serviceName)
	}
}

func (ihrr *InboundHTTPRouteRule) setRateLimit(rateLimit *policyv1alpha1.HTTPPerRouteRateLimitSpec) {
	ihrr.RateLimit = newHTTPPerRouteRateLimit(rateLimit)
}

func (itp *InboundTrafficPolicy) newClusterConfigs(clusterName ClusterName) *WeightedEndpoint {
	if itp.ClustersConfigs == nil {
		itp.ClustersConfigs = make(map[ClusterName]*WeightedEndpoint)
	}
	cluster, exist := itp.ClustersConfigs[clusterName]
	if !exist || cluster == nil {
		newCluster := make(WeightedEndpoint, 0)
		itp.ClustersConfigs[clusterName] = &newCluster
		return &newCluster
	}
	return cluster
}

func (otp *OutboundTrafficPolicy) newClusterConfigs(clusterName ClusterName) *ClusterConfigs {
	if otp.ClustersConfigs == nil {
		otp.ClustersConfigs = make(map[ClusterName]*ClusterConfigs)
	}
	cluster, exist := otp.ClustersConfigs[clusterName]
	if !exist || cluster == nil {
		newCluster := new(ClusterConfigs)
		otp.ClustersConfigs[clusterName] = newCluster
		return newCluster
	}
	return cluster
}

func (otp *ClusterConfigs) addWeightedEndpoint(address Address, port Port, weight Weight) {
	if otp.Endpoints == nil {
		weightedEndpoints := make(WeightedEndpoints)
		otp.Endpoints = &weightedEndpoints
	}
	otp.Endpoints.addWeightedEndpoint(address, port, weight)
}

func (otp *ClusterConfigs) addWeightedZoneEndpoint(address Address, port Port, weight Weight, cluster, lbType, contextPath string) {
	if otp.Endpoints == nil {
		weightedEndpoints := make(WeightedEndpoints)
		otp.Endpoints = &weightedEndpoints
	}
	otp.Endpoints.addWeightedZoneEndpoint(address, port, weight, cluster, lbType, contextPath)
}

func (wes *WeightedEndpoints) addWeightedEndpoint(address Address, port Port, weight Weight) {
	if addrWithPort.MatchString(string(address)) {
		httpHostPort := HTTPHostPort(address)
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight: weight,
		}
	} else {
		httpHostPort := HTTPHostPort(fmt.Sprintf("%s:%d", address, port))
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight: weight,
		}
	}
}

func (wes *WeightedEndpoints) addWeightedZoneEndpoint(address Address, port Port, weight Weight, cluster, lbType, contextPath string) {
	if addrWithPort.MatchString(string(address)) {
		httpHostPort := HTTPHostPort(address)
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight:      weight,
			Cluster:     cluster,
			LBType:      lbType,
			ContextPath: contextPath,
		}
	} else {
		httpHostPort := HTTPHostPort(fmt.Sprintf("%s:%d", address, port))
		(*wes)[httpHostPort] = &WeightedZoneEndpoint{
			Weight:      weight,
			Cluster:     cluster,
			LBType:      lbType,
			ContextPath: contextPath,
		}
	}
}

func (we *WeightedEndpoint) addWeightedEndpoint(address Address, port Port, weight Weight) {
	if addrWithPort.MatchString(string(address)) {
		httpHostPort := HTTPHostPort(address)
		(*we)[httpHostPort] = weight
	} else {
		httpHostPort := HTTPHostPort(fmt.Sprintf("%s:%d", address, port))
		(*we)[httpHostPort] = weight
	}
}

func (otp *ClusterConfigs) setConnectionSettings(connectionSettings *policyv1alpha1.ConnectionSettingsSpec) {
	if connectionSettings == nil {
		otp.ConnectionSettings = nil
		return
	}
	otp.ConnectionSettings = new(ConnectionSettings)
	if connectionSettings.TCP != nil {
		otp.ConnectionSettings.TCP = new(TCPConnectionSettings)
		otp.ConnectionSettings.TCP.MaxConnections = connectionSettings.TCP.MaxConnections
		if connectionSettings.TCP.ConnectTimeout != nil {
			duration := connectionSettings.TCP.ConnectTimeout.Seconds()
			otp.ConnectionSettings.TCP.ConnectTimeout = &duration
		}
	}
	if connectionSettings.HTTP != nil {
		otp.ConnectionSettings.HTTP = new(HTTPConnectionSettings)
		otp.ConnectionSettings.HTTP.MaxRequests = connectionSettings.HTTP.MaxRequests
		otp.ConnectionSettings.HTTP.MaxRequestsPerConnection = connectionSettings.HTTP.MaxRequestsPerConnection
		otp.ConnectionSettings.HTTP.MaxPendingRequests = connectionSettings.HTTP.MaxPendingRequests
		otp.ConnectionSettings.HTTP.MaxRetries = connectionSettings.HTTP.MaxRetries
		if connectionSettings.HTTP.CircuitBreaking != nil {
			otp.ConnectionSettings.HTTP.CircuitBreaking = new(HTTPCircuitBreaking)
			if connectionSettings.HTTP.CircuitBreaking.StatTimeWindow != nil {
				duration := connectionSettings.HTTP.CircuitBreaking.StatTimeWindow.Seconds()
				otp.ConnectionSettings.HTTP.CircuitBreaking.StatTimeWindow = &duration
			}
			otp.ConnectionSettings.HTTP.CircuitBreaking.MinRequestAmount = connectionSettings.HTTP.CircuitBreaking.MinRequestAmount
			if connectionSettings.HTTP.CircuitBreaking.DegradedTimeWindow != nil {
				duration := connectionSettings.HTTP.CircuitBreaking.DegradedTimeWindow.Seconds()
				otp.ConnectionSettings.HTTP.CircuitBreaking.DegradedTimeWindow = &duration
			}
			if connectionSettings.HTTP.CircuitBreaking.SlowTimeThreshold != nil {
				duration := connectionSettings.HTTP.CircuitBreaking.SlowTimeThreshold.Seconds()
				otp.ConnectionSettings.HTTP.CircuitBreaking.SlowTimeThreshold = &duration
			}
			otp.ConnectionSettings.HTTP.CircuitBreaking.SlowAmountThreshold = connectionSettings.HTTP.CircuitBreaking.SlowAmountThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.SlowRatioThreshold = connectionSettings.HTTP.CircuitBreaking.SlowRatioThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.ErrorAmountThreshold = connectionSettings.HTTP.CircuitBreaking.ErrorAmountThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.ErrorRatioThreshold = connectionSettings.HTTP.CircuitBreaking.ErrorRatioThreshold
			otp.ConnectionSettings.HTTP.CircuitBreaking.DegradedStatusCode = connectionSettings.HTTP.CircuitBreaking.DegradedStatusCode
			otp.ConnectionSettings.HTTP.CircuitBreaking.DegradedResponseContent = connectionSettings.HTTP.CircuitBreaking.DegradedResponseContent
		}
	}
}

func (otp *ClusterConfigs) setRetryPolicy(retryPolicy *policyv1alpha1.RetryPolicySpec) {
	if retryPolicy == nil {
		otp.RetryPolicy = nil
		return
	}
	otp.RetryPolicy = new(RetryPolicy)
	otp.RetryPolicy.RetryOn = retryPolicy.RetryOn
	otp.RetryPolicy.NumRetries = retryPolicy.NumRetries
	perTryTimeout := retryPolicy.PerTryTimeout.Seconds()
	otp.RetryPolicy.PerTryTimeout = &perTryTimeout
	retryBackoffBaseInterval := retryPolicy.RetryBackoffBaseInterval.Seconds()
	otp.RetryPolicy.RetryBackoffBaseInterval = &retryBackoffBaseInterval
}

func (ftp *ForwardTrafficPolicy) newForwardMatch(rule string) WeightedClusters {
	if ftp.ForwardMatches == nil {
		ftp.ForwardMatches = make(ForwardTrafficMatches)
	}
	forwardMatch, exist := ftp.ForwardMatches[rule]
	if !exist || forwardMatch == nil {
		forwardMatch = make(WeightedClusters)
		ftp.ForwardMatches[rule] = forwardMatch
		return forwardMatch
	}
	return forwardMatch
}

func (ftp *ForwardTrafficPolicy) newEgressGateway(clusterName ClusterName, mode string) *EgressGatewayClusterConfigs {
	if ftp.EgressGateways == nil {
		ftp.EgressGateways = make(map[ClusterName]*EgressGatewayClusterConfigs)
	}
	cluster, exist := ftp.EgressGateways[clusterName]
	if !exist || cluster == nil {
		newCluster := new(EgressGatewayClusterConfigs)
		newCluster.Mode = mode
		ftp.EgressGateways[clusterName] = newCluster
		return newCluster
	}
	return cluster
}

// Len is the number of elements in the collection.
func (otms OutboundTrafficMatchSlice) Len() int {
	return len(otms)
}

// Less reports whether the element with index i
// must sort before the element with index j.
func (otms OutboundTrafficMatchSlice) Less(i, j int) bool {
	a, b := otms[i], otms[j]

	aLen, bLen := len(a.DestinationIPRanges), len(b.DestinationIPRanges)
	if aLen == 0 && bLen == 0 {
		return false
	}
	if aLen > 0 && bLen == 0 {
		return false
	}
	if aLen == 0 && bLen > 0 {
		return true
	}

	var aCidrs, bCidrs []*cidr.CIDR
	for ipRangea := range a.DestinationIPRanges {
		cidra, _ := cidr.ParseCIDR(string(ipRangea))
		aCidrs = append(aCidrs, cidra)
	}
	for ipRangeb := range b.DestinationIPRanges {
		cidrb, _ := cidr.ParseCIDR(string(ipRangeb))
		bCidrs = append(bCidrs, cidrb)
	}

	cidr.DescSortCIDRs(aCidrs)
	cidr.DescSortCIDRs(bCidrs)

	minLen := aLen
	if aLen > bLen {
		minLen = bLen
	}

	for n := 0; n < minLen; n++ {
		if cidr.CompareCIDR(aCidrs[n], bCidrs[n]) == 1 {
			return true
		}
	}

	return aLen > bLen
}

// Swap swaps the elements with indexes i and j.
func (otms OutboundTrafficMatchSlice) Swap(i, j int) {
	otms[j], otms[i] = otms[i], otms[j]
}

// Sort sorts data.
// It makes one call to data.Len to determine n and O(n*log(n)) calls to
// data.Less and data.Swap. The sort is not guaranteed to be stable.
func (otms *OutboundTrafficMatches) Sort() {
	for _, trafficMatches := range *otms {
		if len(trafficMatches) > 1 {
			sort.Sort(trafficMatches)
		}
	}
}

func (hrrs *OutboundHTTPRouteRuleSlice) sort() {
	if len(*hrrs) > 1 {
		sort.Sort(hrrs)
	}
}

func (hrrs *OutboundHTTPRouteRuleSlice) Len() int {
	return len(*hrrs)
}

func (hrrs *OutboundHTTPRouteRuleSlice) Swap(i, j int) {
	(*hrrs)[j], (*hrrs)[i] = (*hrrs)[i], (*hrrs)[j]
}

func (hrrs *OutboundHTTPRouteRuleSlice) Less(i, j int) bool {
	a, b := (*hrrs)[i], (*hrrs)[j]
	if a.Path == constants.RegexMatchAll {
		return false
	}
	return strings.Compare(string(a.Path), string(b.Path)) == -1
}

func (hrrs *InboundHTTPRouteRules) sort() {
	if len(hrrs.RouteRules) > 1 {
		sort.Sort(hrrs.RouteRules)
	}
}

func (irrs InboundHTTPRouteRuleSlice) Len() int {
	return len(irrs)
}

func (irrs InboundHTTPRouteRuleSlice) Swap(i, j int) {
	irrs[j], irrs[i] = irrs[i], irrs[j]
}

func (irrs InboundHTTPRouteRuleSlice) Less(i, j int) bool {
	a, b := irrs[i], irrs[j]
	if a.Path == constants.RegexMatchAll {
		return false
	}
	return strings.Compare(string(a.Path), string(b.Path)) == -1
}
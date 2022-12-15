package repo

import (
	"encoding/json"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"time"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/sidecar/providers/pipy"
	"github.com/openservicemesh/osm/pkg/sidecar/providers/pipy/client"
)

// PipyConfGeneratorJob is the job to generate pipy policy json
type PipyConfGeneratorJob struct {
	proxy      *pipy.Proxy
	repoServer *Server

	// Optional waiter
	done chan struct{}
}

// GetDoneCh returns the channel, which when closed, indicates the job has been finished.
func (job *PipyConfGeneratorJob) GetDoneCh() <-chan struct{} {
	return job.done
}

// Run is the logic unit of job
func (job *PipyConfGeneratorJob) Run() {
	defer close(job.done)
	if job.proxy == nil {
		return
	}

	s := job.repoServer
	proxy := job.proxy

	proxy.Mutex.Lock()
	defer proxy.Mutex.Unlock()

	proxyServices, err := s.proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Warn().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Msgf("Error looking up services for Sidecar with name=%s", proxy.GetName())
		return
	}

	cataloger := s.catalog
	pipyConf := new(PipyConf)

	probes(proxy, pipyConf)
	features(s, proxy, pipyConf)
	certs(s, proxy, pipyConf)
	plugin(cataloger, proxy.Identity, s, pipyConf, proxy)
	inbound(cataloger, proxy.Identity, s, pipyConf, proxyServices)
	outbound(cataloger, proxy.Identity, s, pipyConf, proxy)
	egress(cataloger, proxy.Identity, s, pipyConf, proxy)
	forward(cataloger, proxy.Identity, s, pipyConf, proxy)
	balance(pipyConf)
	reorder(pipyConf)
	endpoints(pipyConf, s)
	job.publishSidecarConf(s.repoClient, proxy, pipyConf)
}

func endpoints(pipyConf *PipyConf, s *Server) {
	ready := pipyConf.copyAllowedEndpoints(s.kubeController, s.proxyRegistry)
	if !ready {
		if s.retryJob != nil {
			s.retryJob()
		}
	}
}

func balance(pipyConf *PipyConf) {
	pipyConf.rebalancedOutboundClusters()
	pipyConf.rebalancedForwardClusters()
}

func reorder(pipyConf *PipyConf) {
	if pipyConf.Outbound != nil && pipyConf.Outbound.TrafficMatches != nil {
		for _, trafficMatches := range pipyConf.Outbound.TrafficMatches {
			for _, trafficMatch := range trafficMatches {
				for _, routeRules := range trafficMatch.HTTPServiceRouteRules {
					routeRules.RouteRules.sort()
				}
			}
		}
		pipyConf.Outbound.TrafficMatches.Sort()
	}

	if pipyConf.Inbound != nil && pipyConf.Inbound.TrafficMatches != nil {
		for _, trafficMatches := range pipyConf.Inbound.TrafficMatches {
			for _, routeRules := range trafficMatches.HTTPServiceRouteRules {
				routeRules.sort()
			}
		}
	}
}

func egress(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxy *pipy.Proxy) bool {
	egressTrafficPolicy, egressErr := cataloger.GetEgressTrafficPolicy(serviceIdentity)
	if egressErr != nil {
		if s.retryJob != nil {
			s.retryJob()
		}
		return false
	}

	if egressTrafficPolicy != nil {
		egressDependClusters := generatePipyEgressTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf,
			egressTrafficPolicy)
		if len(egressDependClusters) > 0 {
			if ready := generatePipyEgressTrafficBalancePolicy(cataloger, proxy, serviceIdentity, pipyConf,
				egressTrafficPolicy, egressDependClusters); !ready {
				if s.retryJob != nil {
					s.retryJob()
				}
				return false
			}
		}
	}
	return true
}

func forward(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, _ *pipy.Proxy) bool {
	egressGatewayPolicy, egressErr := cataloger.GetEgressGatewayPolicy()
	if egressErr != nil {
		if s.retryJob != nil {
			s.retryJob()
		}
		return false
	}
	if egressGatewayPolicy != nil {
		if ready := generatePipyEgressTrafficForwardPolicy(cataloger, serviceIdentity, pipyConf,
			egressGatewayPolicy); !ready {
			if s.retryJob != nil {
				s.retryJob()
			}
			return false
		}
	}
	return true
}

func outbound(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxy *pipy.Proxy) bool {
	outboundTrafficPolicy := cataloger.GetOutboundMeshTrafficPolicy(serviceIdentity)
	if len(outboundTrafficPolicy.ServicesResolvableSet) > 0 {
		pipyConf.DNSResolveDB = outboundTrafficPolicy.ServicesResolvableSet
	}
	outboundDependClusters := generatePipyOutboundTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf,
		outboundTrafficPolicy)
	if len(outboundDependClusters) > 0 {
		if ready := generatePipyOutboundTrafficBalancePolicy(cataloger, proxy, serviceIdentity, pipyConf,
			outboundTrafficPolicy, outboundDependClusters); !ready {
			if s.retryJob != nil {
				s.retryJob()
			}
			return false
		}
	}
	return true
}

func inbound(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxyServices []service.MeshService) {
	// Build inbound mesh route configurations. These route configurations allow
	// the services associated with this proxy to accept traffic from downstream
	// clients on allowed routes.
	inboundTrafficPolicy := cataloger.GetInboundMeshTrafficPolicy(serviceIdentity, proxyServices)
	generatePipyInboundTrafficPolicy(cataloger, serviceIdentity, pipyConf, inboundTrafficPolicy, s.certManager.GetTrustDomain())
	if len(proxyServices) > 0 {
		for _, svc := range proxyServices {
			if ingressTrafficPolicy, ingressErr := cataloger.GetIngressTrafficPolicy(svc); ingressErr == nil {
				if ingressTrafficPolicy != nil {
					generatePipyIngressTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf, ingressTrafficPolicy)
				}
			}
			if aclTrafficPolicy, aclErr := cataloger.GetAccessControlTrafficPolicy(svc); aclErr == nil {
				if aclTrafficPolicy != nil {
					generatePipyAccessControlTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf, aclTrafficPolicy)
				}
			}
			if expTrafficPolicy, expErr := cataloger.GetExportTrafficPolicy(svc); expErr == nil {
				if expTrafficPolicy != nil {
					generatePipyServiceExportTrafficRoutePolicy(cataloger, serviceIdentity, pipyConf, expTrafficPolicy)
				}
			}
		}
	}
}

func plugin(cataloger catalog.MeshCataloger, serviceIdentity identity.ServiceIdentity, s *Server, pipyConf *PipyConf, proxy *pipy.Proxy) {
	if !s.cfg.GetFeatureFlags().EnablePluginPolicy {
		return
	}

	pluginChains := cataloger.GetPluginChains()
	if len(pluginChains) == 0 {
		return
	}

	pod, err := s.kubeController.GetPodForProxy(proxy)
	if err != nil {
		log.Warn().Str("proxy", proxy.String()).Msg("Could not find pod for connecting proxy.")
		return
	}

	ns := s.kubeController.GetNamespace(pod.Namespace)
	if ns == nil {
		log.Warn().Str("proxy", proxy.String()).Str("namespace", pod.Namespace).Msg("Could not find namespace for connecting proxy.")
	}

	pluginSet := s.pluginSet

	pluginMountedPoints := make(map[string]mapset.Set)

	for _, pluginChain := range pluginChains {
		matched := matchPluginChain(pluginChain, ns, pod)
		if !matched {
			continue
		}
		for _, chain := range pluginChain.Chains {
			for _, pluginName := range chain.Plugins {
				if !pluginSet.Contains(pluginName) {
					log.Warn().Str("proxy", proxy.String()).
						Str("namespace", pod.Namespace).
						Str("plugin", pluginName).
						Msg("Could not find plugin for connecting proxy.")
					continue
				}
				mountedPointSet, exist := pluginMountedPoints[pluginName]
				if !exist {
					mountedPointSet = mapset.NewSet()
					pluginMountedPoints[pluginName] = mountedPointSet
				}
				if !mountedPointSet.Contains(chain.Name) {
					mountedPointSet.Add(chain.Name)
				}
			}
		}
	}

	pipyConf.MountedPlugins = pluginMountedPoints
}

func certs(s *Server, proxy *pipy.Proxy, pipyConf *PipyConf) {
	if mc, ok := s.catalog.(*catalog.MeshCatalog); ok {
		meshConf := mc.GetConfigurator()
		if !(*meshConf).GetSidecarDisabledMTLS() {
			cnPrefix := proxy.Identity.String()
			if proxy.SidecarCert == nil {
				pipyConf.Certificate = nil
				sidecarCert := s.certManager.GetCertificate(cnPrefix)
				if sidecarCert == nil {
					proxy.SidecarCert = nil
				} else {
					proxy.SidecarCert = sidecarCert
				}
			}
			if proxy.SidecarCert == nil || s.certManager.ShouldRotate(proxy.SidecarCert) {
				pipyConf.Certificate = nil
				ct := proxy.PodMetadata.CreationTime
				now := time.Now()
				certValidityPeriod := s.cfg.GetServiceCertValidityPeriod()
				aliveDuration := now.Sub(ct)
				expirationDuration := (aliveDuration + certValidityPeriod/2).Round(certValidityPeriod)
				certExpiration := ct.Add(expirationDuration)
				certValidityPeriod = certExpiration.Sub(now)
				sidecarCert, certErr := s.certManager.IssueCertificate(cnPrefix, certificate.Service, certificate.ValidityDurationProvided(&certValidityPeriod))
				if certErr != nil {
					proxy.SidecarCert = nil
				} else {
					sidecarCert.Expiration = certExpiration
					proxy.SidecarCert = sidecarCert
				}
			}
		} else {
			proxy.SidecarCert = nil
		}
	}
}

func features(s *Server, proxy *pipy.Proxy, pipyConf *PipyConf) {
	if mc, ok := s.catalog.(*catalog.MeshCatalog); ok {
		meshConf := mc.GetConfigurator()
		proxy.MeshConf = meshConf
		pipyConf.setSidecarLogLevel((*meshConf).GetMeshConfig().Spec.Sidecar.LogLevel)
		pipyConf.setEnableSidecarActiveHealthChecks((*meshConf).GetFeatureFlags().EnableSidecarActiveHealthChecks)
		pipyConf.setEnableEgress((*meshConf).IsEgressEnabled())
		pipyConf.setEnablePermissiveTrafficPolicyMode((*meshConf).IsPermissiveTrafficPolicyMode())
		pipyConf.setLocalDNSProxy((*meshConf).IsLocalDNSProxyEnabled(), (*meshConf).GetLocalDNSProxyPrimaryUpstream(), (*meshConf).GetLocalDNSProxySecondaryUpstream())
		clusterProps := (*meshConf).GetMeshConfig().Spec.ClusterSet.Properties
		if len(clusterProps) > 0 {
			pipyConf.Spec.ClusterSet = make(map[string]string)
			for _, prop := range clusterProps {
				pipyConf.Spec.ClusterSet[prop.Name] = prop.Value
			}
		}
	}
}

func probes(proxy *pipy.Proxy, pipyConf *PipyConf) {
	if proxy.PodMetadata != nil {
		if len(proxy.PodMetadata.StartupProbes) > 0 {
			for idx := range proxy.PodMetadata.StartupProbes {
				pipyConf.Spec.Probes.StartupProbes = append(pipyConf.Spec.Probes.StartupProbes, *proxy.PodMetadata.StartupProbes[idx])
			}
		}
		if len(proxy.PodMetadata.LivenessProbes) > 0 {
			for idx := range proxy.PodMetadata.LivenessProbes {
				pipyConf.Spec.Probes.LivenessProbes = append(pipyConf.Spec.Probes.LivenessProbes, *proxy.PodMetadata.LivenessProbes[idx])
			}
		}
		if len(proxy.PodMetadata.ReadinessProbes) > 0 {
			for idx := range proxy.PodMetadata.ReadinessProbes {
				pipyConf.Spec.Probes.ReadinessProbes = append(pipyConf.Spec.Probes.ReadinessProbes, *proxy.PodMetadata.ReadinessProbes[idx])
			}
		}
	}
}

func (job *PipyConfGeneratorJob) publishSidecarConf(repoClient *client.PipyRepoClient, proxy *pipy.Proxy, pipyConf *PipyConf) {
	pipyConf.Ts = nil
	pipyConf.Version = nil
	pipyConf.Certificate = nil
	if proxy.SidecarCert != nil {
		pipyConf.Certificate = &Certificate{
			Expiration: proxy.SidecarCert.Expiration.Format("2006-01-02 15:04:05"),
		}
	}
	bytes, jsonErr := json.Marshal(pipyConf)

	if jsonErr == nil {
		codebasePreV := proxy.ETag

		pluginSetVersion := job.repoServer.pluginSetVersion
		bytes = append(bytes, []byte(pluginSetVersion)...)
		codebaseCurV := hash(bytes)
		if codebaseCurV != codebasePreV {
			codebase := fmt.Sprintf("%s/%s", osmSidecarCodebase, proxy.GetCNPrefix())
			success, err := repoClient.DeriveCodebase(codebase, osmCodebase, codebaseCurV)
			if success {
				ts := time.Now()
				pipyConf.Ts = &ts
				version := fmt.Sprintf("%d", codebaseCurV)
				pipyConf.Version = &version
				if proxy.SidecarCert != nil {
					pipyConf.Certificate.CommonName = &proxy.SidecarCert.CommonName
					pipyConf.Certificate.CertChain = string(proxy.SidecarCert.CertChain)
					pipyConf.Certificate.PrivateKey = string(proxy.SidecarCert.PrivateKey)
					pipyConf.Certificate.IssuingCA = string(proxy.SidecarCert.IssuingCA)
				}
				bytes, _ = json.MarshalIndent(pipyConf, "", " ")
				_, err = repoClient.Batch(fmt.Sprintf("%d", codebaseCurV), []client.Batch{
					{
						Basepath: codebase,
						Items: []client.BatchItem{
							{
								Filename: osmCodebaseConfig,
								Content:  bytes,
							},
						},
					},
				})
			}
			if err != nil {
				log.Error().Err(err)
			} else {
				proxy.ETag = codebaseCurV
			}
		}
	}
}

// JobName implementation for this job, for logging purposes
func (job *PipyConfGeneratorJob) JobName() string {
	return fmt.Sprintf("pipyJob-%s", job.proxy.GetName())
}

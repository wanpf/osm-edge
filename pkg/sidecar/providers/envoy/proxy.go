package envoy

import (
	"fmt"
	"net"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/models"
)

// Proxy is a representation of an Envoy proxy connected to the xDS server.
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {
	// UUID of the proxy
	uuid.UUID

	Identity identity.ServiceIdentity

	net.Addr

	// The time this Proxy connected to the OSM control plane
	connectedAt time.Time

	lastSentVersion    map[TypeURI]uint64
	lastAppliedVersion map[TypeURI]uint64
	lastNonce          map[TypeURI]string

	// Contains the last resource names sent for a given proxy and TypeURL
	lastxDSResourcesSent map[TypeURI]mapset.Set

	// Contains the last requested resource names (and therefore, subscribed) for a given TypeURI
	subscribedResources map[TypeURI]mapset.Set

	// kind is the proxy's kind (ex. sidecar, gateway)
	kind models.ProxyKind

	// Records metadata around the Kubernetes Pod on which this Envoy Proxy is installed.
	// This could be nil if the Envoy is not operating in a Kubernetes cluster (VM for example)
	// NOTE: This field may be not be set at the time Proxy struct is initialized. This would
	// eventually be set when the metadata arrives via the xDS protocol.
	PodMetadata *PodMetadata
}

func (p *Proxy) String() string {
	return fmt.Sprintf("[ProxyUUID=%s], [Pod metadata=%s]", p.UUID, p.PodMetadataString())
}

// PodMetadata is a struct holding information on the Pod on which a given Envoy proxy is installed
// This struct is initialized *eventually*, when the metadata arrives via xDS.
type PodMetadata struct {
	UID            string
	Name           string
	Namespace      string
	IP             string
	ServiceAccount identity.K8sServiceAccount
	Cluster        string
	EnvoyNodeID    string
	WorkloadKind   string
	WorkloadName   string
}

// HasPodMetadata answers the question - has the Pod metadata been recorded for the given Envoy proxy
func (p *Proxy) HasPodMetadata() bool {
	return p.PodMetadata != nil
}

// StatsHeaders returns the headers required for SMI metrics
func (p *Proxy) StatsHeaders() map[string]string {
	unknown := "unknown"
	podName := unknown
	podNamespace := unknown
	podControllerKind := unknown
	podControllerName := unknown

	if p.PodMetadata != nil {
		if len(p.PodMetadata.Name) > 0 {
			podName = p.PodMetadata.Name
		}
		if len(p.PodMetadata.Namespace) > 0 {
			podNamespace = p.PodMetadata.Namespace
		}
		if len(p.PodMetadata.WorkloadKind) > 0 {
			podControllerKind = p.PodMetadata.WorkloadKind
		}
		if len(p.PodMetadata.WorkloadName) > 0 {
			podControllerName = p.PodMetadata.WorkloadName
		}
	}

	// Assume ReplicaSets are controlled by a Deployment unless their names
	// do not contain a hyphen. This aligns with the behavior of the
	// Prometheus config in the OSM Helm chart.
	if podControllerKind == "ReplicaSet" {
		if hyp := strings.LastIndex(podControllerName, "-"); hyp >= 0 {
			podControllerKind = "Deployment"
			podControllerName = podControllerName[:hyp]
		}
	}

	return map[string]string{
		"osm-stats-pod":       podName,
		"osm-stats-namespace": podNamespace,
		"osm-stats-kind":      podControllerKind,
		"osm-stats-name":      podControllerName,
	}
}

// SetLastAppliedVersion records the version of the given Envoy proxy that was last acknowledged.
func (p *Proxy) SetLastAppliedVersion(typeURI TypeURI, version uint64) {
	p.lastAppliedVersion[typeURI] = version
}

// GetLastAppliedVersion returns the last version successfully applied to the given Envoy proxy.
func (p *Proxy) GetLastAppliedVersion(typeURI TypeURI) uint64 {
	return p.lastAppliedVersion[typeURI]
}

// GetLastSentVersion returns the last sent version.
func (p *Proxy) GetLastSentVersion(typeURI TypeURI) uint64 {
	return p.lastSentVersion[typeURI]
}

// IncrementLastSentVersion increments last sent version.
func (p *Proxy) IncrementLastSentVersion(typeURI TypeURI) uint64 {
	p.lastSentVersion[typeURI]++
	return p.GetLastSentVersion(typeURI)
}

// SetLastSentVersion records the version of the given config last sent to the proxy.
func (p *Proxy) SetLastSentVersion(typeURI TypeURI, ver uint64) {
	p.lastSentVersion[typeURI] = ver
}

// GetLastSentNonce returns last sent nonce.
func (p *Proxy) GetLastSentNonce(typeURI TypeURI) string {
	nonce, ok := p.lastNonce[typeURI]
	if !ok {
		p.lastNonce[typeURI] = ""
		return ""
	}
	return nonce
}

// SetNewNonce sets and returns a new nonce.
func (p *Proxy) SetNewNonce(typeURI TypeURI) string {
	p.lastNonce[typeURI] = fmt.Sprintf("%d", time.Now().UnixNano())
	return p.lastNonce[typeURI]
}

// PodMetadataString returns relevant pod metadata as a string
func (p *Proxy) PodMetadataString() string {
	if p.PodMetadata == nil {
		return ""
	}
	return fmt.Sprintf("UID=%s, Namespace=%s, Name=%s, ServiceAccount=%s", p.PodMetadata.UID, p.PodMetadata.Namespace, p.PodMetadata.Name, p.PodMetadata.ServiceAccount.Name)
}

// GetName returns a unique name for this proxy based on the identity and uuid.
func (p *Proxy) GetName() string {
	return fmt.Sprintf("%s:%s", p.Identity.String(), p.UUID.String())
}

// GetUUID returns UUID.
func (p *Proxy) GetUUID() uuid.UUID {
	return p.UUID
}

// GetIdentity returns ServiceIdentity.
func (p *Proxy) GetIdentity() identity.ServiceIdentity {
	return p.Identity
}

// GetConnectedAt returns the timestamp of when the given proxy connected to the control plane.
func (p *Proxy) GetConnectedAt() time.Time {
	return p.connectedAt
}

// GetIP returns the IP address of the Envoy proxy connected to xDS.
func (p *Proxy) GetIP() net.Addr {
	return p.Addr
}

// GetLastResourcesSent returns a set of resources last sent for a proxy givne a TypeURL
// If none were sent, empty set is returned
func (p *Proxy) GetLastResourcesSent(typeURI TypeURI) mapset.Set {
	sentResources, ok := p.lastxDSResourcesSent[typeURI]
	if !ok {
		return mapset.NewSet()
	}
	return sentResources
}

// SetLastResourcesSent sets the last sent resources given a proxy for a TypeURL
func (p *Proxy) SetLastResourcesSent(typeURI TypeURI, resourcesSet mapset.Set) {
	p.lastxDSResourcesSent[typeURI] = resourcesSet
}

// GetSubscribedResources returns a set of resources subscribed for a proxy given a TypeURL
// If none were subscribed, empty set is returned
func (p *Proxy) GetSubscribedResources(typeURI TypeURI) mapset.Set {
	sentResources, ok := p.subscribedResources[typeURI]
	if !ok {
		return mapset.NewSet()
	}
	return sentResources
}

// SetSubscribedResources sets the input resources as subscribed resources given a proxy for a TypeURL
func (p *Proxy) SetSubscribedResources(typeURI TypeURI, resourcesSet mapset.Set) {
	p.subscribedResources[typeURI] = resourcesSet
}

// Kind return the proxy's kind
func (p *Proxy) Kind() models.ProxyKind {
	return p.kind
}

// NewProxy creates a new instance of an Envoy proxy connected to the xDS servers.
func NewProxy(kind models.ProxyKind, uuid uuid.UUID, svcIdentity identity.ServiceIdentity, ip net.Addr) *Proxy {
	return &Proxy{
		// Identity is of the form <name>.<namespace>.cluster.local
		Identity: svcIdentity,
		UUID:     uuid,

		Addr: ip,

		connectedAt: time.Now(),

		lastNonce:            make(map[TypeURI]string),
		lastSentVersion:      make(map[TypeURI]uint64),
		lastAppliedVersion:   make(map[TypeURI]uint64),
		lastxDSResourcesSent: make(map[TypeURI]mapset.Set),
		subscribedResources:  make(map[TypeURI]mapset.Set),

		kind: kind,
	}
}
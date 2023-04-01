// Package server implements OSM CNI Controller.
package server

import "github.com/openservicemesh/osm/pkg/logger"

var (
	log = logger.New("interceptor-ctrl-server")
)

// Server CNI Server.
type Server interface {
	Start() error
	Stop()
}

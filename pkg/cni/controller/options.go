// Package controller implements osm interceptor.
package controller

import (
	log "github.com/sirupsen/logrus"

	"github.com/openservicemesh/osm/pkg/cni/config"
)

// NewOptions setup tasks when start up and return a kubernetes client
func NewOptions() error {
	if config.Debug {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

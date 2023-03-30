package plugin

import (
	"os"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("osm-cni")
)

func init() {
	if logfile, err := os.OpenFile("/tmp/osm-cni.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
		log = log.Output(logfile)
	}
}

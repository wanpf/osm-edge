// Package server implements OSM CNI Controller.
package server

// Server CNI Server.
type Server interface {
	Start() error
	Stop()
}

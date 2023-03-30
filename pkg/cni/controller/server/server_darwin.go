package server

// NewServer returns a new CNI Server.
// the path this the unix path to listen.
func NewServer(unixSockPath string, bpfMountPath string, cniReady, stop chan struct{}) Server {
	panic("unsupported")
}

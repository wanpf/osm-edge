package config

var (
	// Debug indicates debug feature of/off
	Debug = false
	// Skip indicates skip feature of/off
	Skip = false
	// DisableWatcher indicates DisableWatcher feature of/off
	DisableWatcher = false
	// EnableCNI indicates CNI feature enable/disable
	EnableCNI = false
	// IsKind indicates Kubernetes running in Docker
	IsKind = false
	// HostProc defines HostProc volume
	HostProc string
	// CNIBinDir defines CNIBIN volume
	CNIBinDir string
	// CNIConfigDir defines CNIConfig volume
	CNIConfigDir string
	// HostVarRun defines HostVar volume
	HostVarRun string
	// KubeConfig defines kube config
	KubeConfig string
	// Context defines kube context
	Context string
	// EnableHotRestart indicates HotRestart feature enable/disable
	EnableHotRestart = false
)

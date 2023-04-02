// Package main implements osm intercepter.
package main

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/openservicemesh/osm/pkg/cni/config"
	"github.com/openservicemesh/osm/pkg/cni/controller"
	"github.com/openservicemesh/osm/pkg/cni/controller/helpers"
	cniserver "github.com/openservicemesh/osm/pkg/cni/controller/server"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("osm-interceptor-cli")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "osm-interceptor",
	Short: "Use eBPF to speed up your Service Mesh like crossing an Einstein-Rosen Bridge.",
	Long:  `Use eBPF to speed up your Service Mesh like crossing an Einstein-Rosen Bridge.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := helpers.LoadProgs(config.EnableCNI, config.KernelTracing); err != nil {
			return fmt.Errorf("failed to load ebpf programs: %v", err)
		}

		stop := make(chan struct{}, 1)
		cniReady := make(chan struct{}, 1)
		if config.EnableCNI {
			s := cniserver.NewServer(path.Join("/host", config.CNISock), "/sys/fs/bpf", cniReady, stop)
			if err := s.Start(); err != nil {
				log.Fatal().Err(err)
				return err
			}
		}
		// todo: wait for stop
		if err := controller.Run(cniReady, stop); err != nil {
			log.Fatal().Err(err)
			return err
		}
		return nil
	},
}

func execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func main() {
	execute()
}

func init() {
	// Get some flags from commands
	rootCmd.PersistentFlags().BoolVarP(&config.KernelTracing, "kernel-tracing", "d", false, "KernelTracing mode")
	rootCmd.PersistentFlags().BoolVarP(&config.IsKind, "kind", "k", false, "Enable when Kubernetes is running in Kind")
	rootCmd.PersistentFlags().BoolVar(&config.EnableCNI, "cni-mode", false, "Enable CNI plugin")
	rootCmd.PersistentFlags().StringVar(&config.HostProc, "host-proc", "/host/proc", "/proc mount path")
	rootCmd.PersistentFlags().StringVar(&config.CNIBinDir, "cni-bin-dir", "/host/opt/cni/bin", "/opt/cni/bin mount path")
	rootCmd.PersistentFlags().StringVar(&config.CNIConfigDir, "cni-config-dir", "/host/etc/cni/net.d", "/etc/cni/net.d mount path")
	rootCmd.PersistentFlags().StringVar(&config.HostVarRun, "host-var-run", "/host/var/run", "/var/run mount path")
	rootCmd.PersistentFlags().StringVar(&config.KubeConfig, "kubeconfig", "", "Kubernetes configuration file")
	rootCmd.PersistentFlags().StringVar(&config.Context, "kubecontext", "", "The name of the kube config context to use")
}

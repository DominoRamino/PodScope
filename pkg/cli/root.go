package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "podscope",
	Short: "PodScope - Kubernetes pod network traffic analyzer",
	Long: `PodScope Shark is a lightweight, ephemeral network traffic capture tool
for Kubernetes. It attaches to pods using ephemeral containers to capture
and analyze network traffic without modifying your deployments.

Features:
  - Zero-intrusion packet capture via ephemeral containers
  - HTTP/1.1 plaintext traffic analysis
  - TLS handshake metadata extraction (SNI, timing)
  - Real-time traffic visualization
  - PCAP export for Wireshark analysis`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(tapCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("podscope version %s\n", Version)
	},
}

func exitWithError(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
	os.Exit(1)
}

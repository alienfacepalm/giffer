package cli

import (
	"github.com/AlienFacepalm/giffer/internal/ui"
	"github.com/spf13/cobra"
)

func newUICmd() *cobra.Command {
	var (
		addr        string
		uploadDir   string
		allowRemote bool
	)

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "🖥️  Start the local convert UI in a native window",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDesktopUIWith(cmd, addr, uploadDir, allowRemote)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8765", "listen address (loopback only unless --allow-remote; never remapped)")
	cmd.Flags().StringVar(&uploadDir, "upload-dir", ui.DefaultUploadDir(), "directory for uploaded zips and GIF output")
	cmd.Flags().BoolVar(&allowRemote, "allow-remote", false, "allow non-loopback --addr (exposes convert API on the network)")
	return cmd
}

func runDesktopUI(cmd *cobra.Command) error {
	return runDesktopUIWith(cmd, "127.0.0.1:8765", ui.DefaultUploadDir(), false)
}

func runDesktopUIWith(cmd *cobra.Command, addr, uploadDir string, allowRemote bool) error {
	return ui.Run(ui.Options{Addr: addr, UploadDir: uploadDir, AllowRemote: allowRemote}, cmd.OutOrStdout())
}

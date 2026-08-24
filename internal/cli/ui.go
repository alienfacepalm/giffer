package cli

import (
	"github.com/AlienFacepalm/giffer/internal/ui"
	"github.com/spf13/cobra"
)

func newUICmd() *cobra.Command {
	var (
		addr      string
		uploadDir string
	)

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "🖥️  Start the local convert UI in a native window",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDesktopUIWith(cmd, addr, uploadDir)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8765", "listen address (fixed; never remapped)")
	cmd.Flags().StringVar(&uploadDir, "upload-dir", ui.DefaultUploadDir(), "directory for uploaded zips and GIF output")
	return cmd
}

func runDesktopUI(cmd *cobra.Command) error {
	return runDesktopUIWith(cmd, "127.0.0.1:8765", ui.DefaultUploadDir())
}

func runDesktopUIWith(cmd *cobra.Command, addr, uploadDir string) error {
	return ui.Run(ui.Options{Addr: addr, UploadDir: uploadDir}, cmd.OutOrStdout())
}

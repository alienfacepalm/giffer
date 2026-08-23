package cli

import (
	"fmt"

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
		Short: "Start the local convert UI in a browser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "giffer ui listening on http://%s\n", addr)
			fmt.Fprintf(cmd.OutOrStdout(), "upload dir: %s\n", uploadDir)
			srv := ui.New(ui.Options{Addr: addr, UploadDir: uploadDir})
			return srv.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8765", "listen address")
	cmd.Flags().StringVar(&uploadDir, "upload-dir", defaultUploadDir, "directory for uploaded zips and GIF output")
	return cmd
}

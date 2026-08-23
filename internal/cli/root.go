package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlienFacepalm/giffer/internal/convert"
	"github.com/spf13/cobra"
)

const (
	exitOK            = 0
	exitRuntime       = 1
	exitInvalidParams = 2
)

// Execute runs the root command with process args and returns an exit code.
func Execute() int {
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}

// Run executes giffer with the given args and I/O streams (for tests).
func Run(args []string, stdout, stderr io.Writer) int {
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		var inv *invalidParamsError
		if errors.As(err, &inv) {
			fmt.Fprintln(stderr, inv.Error())
			return exitInvalidParams
		}
		fmt.Fprintln(stderr, err.Error())
		return exitRuntime
	}
	return exitOK
}

type invalidParamsError struct {
	msg string
}

func (e *invalidParamsError) Error() string { return e.msg }

func newRootCmd() *cobra.Command {
	var (
		input    string
		output   string
		delayMS  int
		maxWidth int
		loop     int
	)

	cmd := &cobra.Command{
		Use:           "giffer",
		Short:         "Convert a zip of photos into an animated GIF",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := validateOptions(input, output, delayMS, maxWidth, loop)
			if err != nil {
				return err
			}

			if _, err := os.Stat(opts.Output); err == nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: overwriting %s\n", opts.Output)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("check output path: %w", err)
			}

			if err := convert.ZipToGIF(opts); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), opts.Output)
			return nil
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "path to input .zip (e.g. upload/photos.zip)")
	cmd.Flags().StringVar(&output, "output", "", "destination .gif path (default: beside the zip)")
	cmd.Flags().IntVar(&delayMS, "delay-ms", 500, "milliseconds each frame is shown")
	cmd.Flags().IntVar(&maxWidth, "max-width", 800, "max frame width in px; height scales to preserve aspect")
	cmd.Flags().IntVar(&loop, "loop", 0, "GIF loop count; 0 means loop forever")

	return cmd
}

func validateOptions(input, output string, delayMS, maxWidth, loop int) (convert.Options, error) {
	if strings.TrimSpace(input) == "" {
		return convert.Options{}, &invalidParamsError{msg: "--input is required"}
	}
	if !strings.EqualFold(filepath.Ext(input), ".zip") {
		return convert.Options{}, &invalidParamsError{msg: "--input must be a .zip file"}
	}
	if delayMS <= 0 {
		return convert.Options{}, &invalidParamsError{msg: "--delay-ms must be an integer > 0"}
	}
	if maxWidth < 1 {
		return convert.Options{}, &invalidParamsError{msg: "--max-width must be an integer >= 1"}
	}
	if loop < 0 {
		return convert.Options{}, &invalidParamsError{msg: "--loop must be an integer >= 0"}
	}

	out := strings.TrimSpace(output)
	if out == "" {
		base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
		out = filepath.Join(filepath.Dir(input), base+".gif")
	}
	if !strings.EqualFold(filepath.Ext(out), ".gif") {
		return convert.Options{}, &invalidParamsError{msg: "--output must end in .gif"}
	}

	return convert.Options{
		Input:    input,
		Output:   out,
		DelayMS:  delayMS,
		MaxWidth: maxWidth,
		Loop:     loop,
	}, nil
}

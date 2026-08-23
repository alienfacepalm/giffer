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
		inputs   []string
		output   string
		delayMS  int
		maxWidth int
		loop     int
	)

	cmd := &cobra.Command{
		Use:   "giffer",
		Short: "Convert photos (zip, files, or directories) into an animated GIF",
		Long: `Convert one or more inputs into a single animated GIF.

Inputs may be .zip archives, individual image files (jpg/png/webp/gif),
or directories of images. Repeat --input and/or pass paths as arguments.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			allInputs := append([]string{}, inputs...)
			allInputs = append(allInputs, args...)

			opts, err := validateOptions(allInputs, output, delayMS, maxWidth, loop)
			if err != nil {
				return err
			}

			if _, err := os.Stat(opts.Output); err == nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: overwriting %s\n", opts.Output)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("check output path: %w", err)
			}

			if err := convert.ToGIF(opts); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), opts.Output)
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&inputs, "input", nil, "zip, image file, or directory (repeatable)")
	cmd.Flags().StringVar(&output, "output", "", "destination .gif path (default depends on inputs)")
	cmd.Flags().IntVar(&delayMS, "delay-ms", 500, "milliseconds each frame is shown")
	cmd.Flags().IntVar(&maxWidth, "max-width", 800, "max frame width in px; height scales to preserve aspect")
	cmd.Flags().IntVar(&loop, "loop", 0, "GIF loop count; 0 means loop forever")

	return cmd
}

func validateOptions(inputs []string, output string, delayMS, maxWidth, loop int) (convert.Options, error) {
	cleaned := make([]string, 0, len(inputs))
	for _, in := range inputs {
		in = strings.TrimSpace(in)
		if in == "" {
			continue
		}
		cleaned = append(cleaned, in)
	}
	if len(cleaned) == 0 {
		return convert.Options{}, &invalidParamsError{msg: "at least one input is required (--input and/or path arguments)"}
	}

	for _, in := range cleaned {
		if err := validateInputPath(in); err != nil {
			return convert.Options{}, err
		}
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
		out = defaultOutput(cleaned)
	}
	if !strings.EqualFold(filepath.Ext(out), ".gif") {
		return convert.Options{}, &invalidParamsError{msg: "--output must end in .gif"}
	}

	return convert.Options{
		Inputs:   cleaned,
		Output:   out,
		DelayMS:  delayMS,
		MaxWidth: maxWidth,
		Loop:     loop,
	}, nil
}

func validateInputPath(in string) error {
	info, err := os.Stat(in)
	if err != nil {
		// Allow missing paths through to convert for consistent runtime errors,
		// except clearly invalid extensions when the path does not exist as a dir.
		if os.IsNotExist(err) {
			ext := strings.ToLower(filepath.Ext(in))
			switch ext {
			case ".zip", ".jpg", ".jpeg", ".png", ".webp", ".gif":
				return nil
			case "":
				return nil // may be a directory that will fail at convert time
			default:
				return &invalidParamsError{msg: fmt.Sprintf("unsupported input %q: use a .zip, image file, or directory", in)}
			}
		}
		return &invalidParamsError{msg: fmt.Sprintf("cannot access input %q: %v", in, err)}
	}
	if info.IsDir() {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(in))
	switch ext {
	case ".zip", ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return nil
	default:
		return &invalidParamsError{msg: fmt.Sprintf("unsupported input %q: use a .zip, image file, or directory", in)}
	}
}

func defaultOutput(inputs []string) string {
	if len(inputs) == 1 {
		in := inputs[0]
		info, err := os.Stat(in)
		if err == nil && info.IsDir() {
			base := filepath.Base(in)
			if base == "." || base == string(filepath.Separator) || base == "" {
				base = "animation"
			}
			return filepath.Join(filepath.Dir(in), base+".gif")
		}
		base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
		if base == "" {
			base = "animation"
		}
		return filepath.Join(filepath.Dir(in), base+".gif")
	}
	return "animation.gif"
}

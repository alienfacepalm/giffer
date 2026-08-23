package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/AlienFacepalm/giffer/internal/convert"
	"github.com/spf13/cobra"
)

const (
	exitOK            = 0
	exitRuntime       = 1
	exitInvalidParams = 2

	defaultUploadDir = "upload"
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
		Short:         "Convert zips or photo directories into animated GIFs",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateTunables(delayMS, maxWidth, loop); err != nil {
				return err
			}

			if strings.TrimSpace(input) == "" {
				return runBatch(cmd, defaultUploadDir, delayMS, maxWidth, loop)
			}

			opts, err := validateSingleOptions(input, output, delayMS, maxWidth, loop)
			if err != nil {
				return err
			}

			if _, err := os.Stat(opts.Output); err == nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: overwriting %s\n", opts.Output)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("check output path: %w", err)
			}

			if err := convert.Convert(opts); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), opts.Output)
			return nil
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "path to input .zip or photo directory (omit to process all of upload/)")
	cmd.Flags().StringVar(&output, "output", "", "destination .gif path (default: beside the input)")
	cmd.Flags().IntVar(&delayMS, "delay-ms", 500, "milliseconds each frame is shown")
	cmd.Flags().IntVar(&maxWidth, "max-width", 800, "max frame width in px; height scales to preserve aspect")
	cmd.Flags().IntVar(&loop, "loop", 0, "GIF loop count; 0 means loop forever")

	cmd.AddCommand(newUICmd())
	return cmd
}

type batchJob struct {
	Input  string
	Output string
}

func runBatch(cmd *cobra.Command, uploadDir string, delayMS, maxWidth, loop int) error {
	jobs, err := discoverUploadJobs(uploadDir)
	if err != nil {
		return err
	}

	var toRun []batchJob
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	for _, j := range jobs {
		if _, err := os.Stat(j.Output); err == nil {
			fmt.Fprintf(stdout, "skip %s\n", j.Output)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check output path %s: %w", j.Output, err)
		}
		toRun = append(toRun, j)
	}

	if len(toRun) == 0 {
		return nil
	}

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed int

	for _, j := range toRun {
		wg.Add(1)
		go func(j batchJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := convert.Convert(convert.Options{
				Input:    j.Input,
				Output:   j.Output,
				DelayMS:  delayMS,
				MaxWidth: maxWidth,
				Loop:     loop,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				fmt.Fprintf(stderr, "%s: %v\n", j.Input, err)
				return
			}
			fmt.Fprintln(stdout, j.Output)
		}(j)
	}
	wg.Wait()

	if failed > 0 {
		return fmt.Errorf("%d conversion(s) failed", failed)
	}
	return nil
}

func discoverUploadJobs(uploadDir string) ([]batchJob, error) {
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("upload directory %q: %w", uploadDir, err)
	}

	byOutput := make(map[string]string) // lower(output) -> input
	var jobs []batchJob
	var collisions []string

	addJob := func(input, output string) {
		key := strings.ToLower(filepath.Clean(output))
		if prev, ok := byOutput[key]; ok {
			collisions = append(collisions, fmt.Sprintf("%s and %s both target %s", prev, input, output))
			return
		}
		byOutput[key] = input
		jobs = append(jobs, batchJob{Input: input, Output: output})
	}

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(uploadDir, name)
		ext := filepath.Ext(name)

		if strings.EqualFold(ext, ".gif") {
			continue
		}

		if strings.EqualFold(ext, ".zip") {
			base := strings.TrimSuffix(name, ext)
			addJob(full, filepath.Join(uploadDir, base+".gif"))
			continue
		}

		if !e.IsDir() {
			continue
		}
		if !convert.HasImages(full) {
			continue
		}
		addJob(full, filepath.Join(uploadDir, name+".gif"))
	}

	if len(collisions) > 0 {
		return nil, fmt.Errorf("output collision: %s", strings.Join(collisions, "; "))
	}
	return jobs, nil
}

func validateTunables(delayMS, maxWidth, loop int) error {
	if delayMS <= 0 {
		return &invalidParamsError{msg: "--delay-ms must be an integer > 0"}
	}
	if maxWidth < 1 {
		return &invalidParamsError{msg: "--max-width must be an integer >= 1"}
	}
	if loop < 0 {
		return &invalidParamsError{msg: "--loop must be an integer >= 0"}
	}
	return nil
}

func validateSingleOptions(input, output string, delayMS, maxWidth, loop int) (convert.Options, error) {
	if err := validateTunables(delayMS, maxWidth, loop); err != nil {
		return convert.Options{}, err
	}

	in := strings.TrimSpace(input)
	if in == "" {
		return convert.Options{}, &invalidParamsError{msg: "--input is required"}
	}

	info, err := os.Stat(in)
	if err != nil {
		if strings.EqualFold(filepath.Ext(in), ".zip") {
			// Let Convert surface unreadable zip; still validate shape.
		} else if errors.Is(err, os.ErrNotExist) {
			return convert.Options{}, &invalidParamsError{msg: "--input must be a .zip file or an existing directory"}
		} else {
			return convert.Options{}, fmt.Errorf("check input path: %w", err)
		}
	} else if info.IsDir() {
		// directory input OK
	} else if !strings.EqualFold(filepath.Ext(in), ".zip") {
		return convert.Options{}, &invalidParamsError{msg: "--input must be a .zip file or an existing directory"}
	}

	out := strings.TrimSpace(output)
	if out == "" {
		base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
		if info != nil && info.IsDir() {
			base = filepath.Base(in)
		}
		out = filepath.Join(filepath.Dir(in), base+".gif")
	}
	if !strings.EqualFold(filepath.Ext(out), ".gif") {
		return convert.Options{}, &invalidParamsError{msg: "--output must end in .gif"}
	}

	return convert.Options{
		Input:    in,
		Output:   out,
		DelayMS:  delayMS,
		MaxWidth: maxWidth,
		Loop:     loop,
	}, nil
}

// validateOptions kept for older tests; delegates to single-input validation.
func validateOptions(input, output string, delayMS, maxWidth, loop int) (convert.Options, error) {
	return validateSingleOptions(input, output, delayMS, maxWidth, loop)
}

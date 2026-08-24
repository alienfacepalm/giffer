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
	return Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

// Run executes giffer with the given args and I/O streams (for tests).
// Pass a non-terminal stdin (for example bytes.NewReader(nil)) to skip the
// interactive wizard when no flags are set.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		var inv *invalidParamsError
		if errors.As(err, &inv) {
			fmt.Fprintln(stderr, fancyError(stderr, inv.Error()))
			return exitInvalidParams
		}
		fmt.Fprintln(stderr, fancyError(stderr, convert.UserMessage(err)))
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
		Short:         "🎞️  Convert photo archives or directories into animated GIFs",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if noUserParams(cmd) && isTerminal(cmd.InOrStdin()) {
				return runWizard(cmd, cmd.InOrStdin(), cmd.OutOrStdout())
			}

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
				printOverwriteWarning(cmd.ErrOrStderr(), opts.Output)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("check output path: %w", err)
			}

			prog := newProgress(cmd.ErrOrStderr(), filepath.Base(opts.Input))
			opts.OnProgress = prog.handler()
			if err := convert.Convert(opts); err != nil {
				prog.finish(false)
				return err
			}
			prog.finish(true)

			printDone(cmd.OutOrStdout(), opts.Output)
			return nil
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "path to photo archive or directory (omit for batch, or run with no flags for the wizard)")
	cmd.Flags().StringVar(&output, "output", "", "destination .gif path (default: beside the input)")
	cmd.Flags().IntVar(&delayMS, "delay-ms", 100, "milliseconds each frame is shown (GIF-safe default; stored in 10ms units)")
	cmd.Flags().IntVar(&maxWidth, "max-width", 0, "max frame width in px; 0 = first photo width")
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
	skipped := 0

	for _, j := range jobs {
		if _, err := os.Stat(j.Output); err == nil {
			printSkip(stdout, j.Output)
			skipped++
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check output path %s: %w", j.Output, err)
		}
		toRun = append(toRun, j)
	}

	printBatchHeader(stderr, len(toRun), skipped)

	if len(toRun) == 0 {
		printBatchSummary(stderr, 0, 0, skipped)
		return nil
	}

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed, okCount int

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
				fmt.Fprintf(stderr, "%s %s: %s\n", newStyle(stderr).red("❌"), j.Input, convert.UserMessage(err))
				return
			}
			okCount++
			printDone(stdout, j.Output)
		}(j)
	}
	wg.Wait()

	printBatchSummary(stderr, okCount, failed, skipped)

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

		if strings.EqualFold(filepath.Ext(name), ".gif") {
			continue
		}

		if convert.IsArchive(name) {
			stem := convert.ArchiveStem(name)
			addJob(full, filepath.Join(uploadDir, stem+".gif"))
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
	if maxWidth < 0 {
		return &invalidParamsError{msg: "--max-width must be an integer >= 0"}
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
		if convert.IsArchive(in) {
			// Let Convert surface unreadable archive; still validate shape.
		} else if errors.Is(err, os.ErrNotExist) {
			return convert.Options{}, &invalidParamsError{msg: "--input must be a photo archive or an existing directory"}
		} else {
			return convert.Options{}, fmt.Errorf("check input path: %w", err)
		}
	} else if info.IsDir() {
		// directory input OK
	} else if !convert.IsArchive(in) {
		return convert.Options{}, &invalidParamsError{
			msg: "--input must be a photo archive (" + strings.Join(convert.ArchiveKinds, ", ") + ") or an existing directory",
		}
	}

	out := strings.TrimSpace(output)
	if out == "" {
		base := convert.ArchiveStem(in)
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

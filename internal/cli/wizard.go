package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlienFacepalm/giffer/internal/convert"
	"github.com/spf13/cobra"
)

// runWizard prompts for convert options when the user runs giffer with no flags.
func runWizard(cmd *cobra.Command, in io.Reader, out io.Writer) error {
	err := runWizardSteps(cmd, in, out)
	if errors.Is(err, io.EOF) {
		printCancelled(out)
		return nil
	}
	return err
}

func runWizardSteps(cmd *cobra.Command, in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	s := newStyle(out)

	printBanner(out)

	mode, err := promptChoice(r, out, "🎮 Mode", []string{
		"📦 Batch convert everything in upload/",
		"🎯 Convert a single archive or directory",
	}, 1)
	if err != nil {
		return err
	}

	delayMS, err := promptInt(r, out, "⏱️  Frame delay (ms)", 100, func(n int) error {
		if n <= 0 {
			return errors.New("must be > 0")
		}
		return nil
	})
	if err != nil {
		return err
	}
	maxWidth, err := promptInt(r, out, "📐 Max width px (0 = first photo width)", 0, func(n int) error {
		if n < 0 {
			return errors.New("must be >= 0")
		}
		return nil
	})
	if err != nil {
		return err
	}
	loop, err := promptInt(r, out, "🔁 Loop count (0 = forever)", 0, func(n int) error {
		if n < 0 {
			return errors.New("must be >= 0")
		}
		return nil
	})
	if err != nil {
		return err
	}

	if mode == 1 {
		jobs, err := discoverUploadJobs(defaultUploadDir)
		if err != nil {
			return err
		}
		pending := 0
		for _, j := range jobs {
			if _, err := os.Stat(j.Output); errors.Is(err, os.ErrNotExist) {
				pending++
			}
		}
		fmt.Fprintf(out, "\n%s Found %s source(s); %s to convert %s\n",
			s.cyan("🔍"),
			s.bold(fmt.Sprintf("%d", len(jobs))),
			s.bold(fmt.Sprintf("%d", pending)),
			s.dim("(existing GIFs are skipped)"),
		)
		ok, err := promptYesNo(r, out, "🚀 Run batch convert?", true)
		if err != nil {
			return err
		}
		if !ok {
			printCancelled(out)
			return nil
		}
		return runBatch(cmd, defaultUploadDir, delayMS, maxWidth, loop)
	}

	suggest := ""
	if jobs, err := discoverUploadJobs(defaultUploadDir); err == nil && len(jobs) > 0 {
		fmt.Fprintf(out, "\n%s Sources in upload/:\n", s.cyan("📁"))
		for i, j := range jobs {
			fmt.Fprintf(out, "  %s %d) %s\n", s.dim("•"), i+1, j.Input)
		}
		suggest = jobs[0].Input
	}

	input, err := promptString(r, out, "📥 Input path (archive or directory)", suggest)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input) == "" {
		return &invalidParamsError{msg: "input is required"}
	}

	defOut := ""
	if opts, err := validateSingleOptions(input, "", delayMS, maxWidth, loop); err == nil {
		defOut = opts.Output
	} else {
		base := convert.ArchiveStem(input)
		if info, statErr := os.Stat(input); statErr == nil && info.IsDir() {
			base = filepath.Base(input)
		}
		defOut = filepath.Join(filepath.Dir(input), base+".gif")
	}

	output, err := promptString(r, out, "📤 Output .gif path", defOut)
	if err != nil {
		return err
	}

	opts, err := validateSingleOptions(input, output, delayMS, maxWidth, loop)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\n%s Convert %s %s %s\n",
		s.magenta("🎬"),
		s.bold(opts.Input),
		s.dim("→"),
		s.bold(opts.Output),
	)
	ok, err := promptYesNo(r, out, "✨ Proceed?", true)
	if err != nil {
		return err
	}
	if !ok {
		printCancelled(out)
		return nil
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
	printDone(out, opts.Output)
	return nil
}

func noUserParams(cmd *cobra.Command) bool {
	for _, name := range []string{"input", "output", "delay-ms", "max-width", "loop"} {
		if cmd.Flags().Changed(name) {
			return false
		}
	}
	return true
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return fileIsTerminal(f)
}

func isWriterTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return fileIsTerminal(f)
}

func fileIsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func promptChoice(r *bufio.Reader, out io.Writer, label string, options []string, def int) (int, error) {
	s := newStyle(out)
	fmt.Fprintf(out, "%s\n", s.bold(label))
	for i, opt := range options {
		marker := s.dim(fmt.Sprintf("%d)", i+1))
		if i+1 == def {
			marker = s.cyan(fmt.Sprintf("%d)", i+1))
		}
		fmt.Fprintf(out, "  %s %s\n", marker, opt)
	}
	for {
		fmt.Fprintf(out, "%sChoice [%d]: ", promptPrefix(out, "👉"), def)
		line, err := readLine(r)
		if err != nil {
			return 0, err
		}
		if line == "" {
			return def, nil
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(options) {
			fmt.Fprintf(out, "%s Enter a number from 1 to %d.\n", s.yellow("🤔"), len(options))
			continue
		}
		return n, nil
	}
}

func promptString(r *bufio.Reader, out io.Writer, label, def string) (string, error) {
	s := newStyle(out)
	if def != "" {
		fmt.Fprintf(out, "%s%s [%s]: ", promptPrefix(out, ""), s.bold(label), s.dim(def))
	} else {
		fmt.Fprintf(out, "%s%s: ", promptPrefix(out, ""), s.bold(label))
	}
	line, err := readLine(r)
	if err != nil {
		return "", err
	}
	if line == "" {
		return def, nil
	}
	return line, nil
}

func promptInt(r *bufio.Reader, out io.Writer, label string, def int, validate func(int) error) (int, error) {
	s := newStyle(out)
	for {
		fmt.Fprintf(out, "%s%s [%s]: ", promptPrefix(out, ""), s.bold(label), s.dim(fmt.Sprintf("%d", def)))
		line, err := readLine(r)
		if err != nil {
			return 0, err
		}
		if line == "" {
			if err := validate(def); err != nil {
				fmt.Fprintf(out, "%s %v\n", s.yellow("🤔"), err)
				continue
			}
			return def, nil
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			fmt.Fprintf(out, "%s Enter an integer.\n", s.yellow("🤔"))
			continue
		}
		if err := validate(n); err != nil {
			fmt.Fprintf(out, "%s %v\n", s.yellow("🤔"), err)
			continue
		}
		return n, nil
	}
}

func promptYesNo(r *bufio.Reader, out io.Writer, label string, defYes bool) (bool, error) {
	s := newStyle(out)
	hint := "Y/n"
	if !defYes {
		hint = "y/N"
	}
	for {
		fmt.Fprintf(out, "%s%s [%s]: ", promptPrefix(out, ""), s.bold(label), s.dim(hint))
		line, err := readLine(r)
		if err != nil {
			return false, err
		}
		if line == "" {
			return defYes, nil
		}
		switch strings.ToLower(line) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Fprintf(out, "%s Enter y or n.\n", s.yellow("🤔"))
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" && errors.Is(err, io.EOF) {
		return "", io.EOF
	}
	return line, nil
}

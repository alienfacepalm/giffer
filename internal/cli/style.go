package cli

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/AlienFacepalm/giffer/internal/convert"
)

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiCyan    = "\033[36m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiMagenta = "\033[35m"
	ansiRed     = "\033[31m"
)

var enableANSIOnce sync.Once

// styleWriter wraps an io.Writer with optional ANSI styling when it's a TTY.
type styleWriter struct {
	w     io.Writer
	color bool
}

func newStyle(w io.Writer) styleWriter {
	enableANSIOnce.Do(enableWindowsANSI)
	return styleWriter{w: w, color: isWriterTerminal(w)}
}

func (s styleWriter) paint(code, text string) string {
	if !s.color || text == "" {
		return text
	}
	return code + text + ansiReset
}

func (s styleWriter) bold(text string) string    { return s.paint(ansiBold, text) }
func (s styleWriter) dim(text string) string     { return s.paint(ansiDim, text) }
func (s styleWriter) cyan(text string) string    { return s.paint(ansiCyan, text) }
func (s styleWriter) green(text string) string   { return s.paint(ansiGreen, text) }
func (s styleWriter) yellow(text string) string  { return s.paint(ansiYellow, text) }
func (s styleWriter) magenta(text string) string { return s.paint(ansiMagenta, text) }
func (s styleWriter) red(text string) string     { return s.paint(ansiRed, text) }

func printBanner(out io.Writer) {
	s := newStyle(out)
	top := "╭──────────────────────────────────────────────╮"
	bot := "╰──────────────────────────────────────────────╯"
	fmt.Fprintln(out, s.magenta(top))
	fmt.Fprintln(out, s.magenta("│")+"  "+s.bold(s.cyan("✨ giffer"))+"  "+s.dim("· photos → animated GIF")+"          "+s.magenta("│"))
	fmt.Fprintln(out, s.magenta("│")+"  "+s.dim("🎞️  sort · resize · encode · loop")+"             "+s.magenta("│"))
	fmt.Fprintln(out, s.magenta(bot))
	fmt.Fprintln(out)
}

func printCancelled(out io.Writer) {
	s := newStyle(out)
	fmt.Fprintln(out, s.yellow("👋 Cancelled."))
}

func printOverwriteWarning(w io.Writer, path string) {
	s := newStyle(w)
	fmt.Fprintf(w, "%s warning: overwriting %s\n", s.yellow("⚠️"), path)
}

func printSkip(out io.Writer, path string) {
	s := newStyle(out)
	fmt.Fprintf(out, "%sskip %s\n", s.dim("⏭️  "), path)
}

func fancyError(w io.Writer, msg string) string {
	s := newStyle(w)
	return s.red("❌") + " " + msg
}

func printDone(out io.Writer, path string) {
	s := newStyle(out)
	if isWriterTerminal(out) {
		fmt.Fprintf(out, "%s %s\n", s.green("✅"), path)
		return
	}
	// Machine-readable: path alone (tests / scripts parse stdout).
	fmt.Fprintln(out, path)
}

func printBatchHeader(out io.Writer, pending, skipped int) {
	if !isWriterTerminal(out) {
		return
	}
	s := newStyle(out)
	fmt.Fprintf(out, "%s batch: %s to convert, %s already done\n",
		s.cyan("📦"),
		s.bold(fmt.Sprintf("%d", pending)),
		s.dim(fmt.Sprintf("%d", skipped)),
	)
}

func printBatchSummary(out io.Writer, ok, failed, skipped int) {
	if !isWriterTerminal(out) {
		return
	}
	s := newStyle(out)
	fmt.Fprintln(out)
	if failed > 0 {
		fmt.Fprintf(out, "%s done with errors — %s ok, %s failed, %s skipped\n",
			s.red("💥"),
			s.green(fmt.Sprintf("%d", ok)),
			s.red(fmt.Sprintf("%d", failed)),
			s.dim(fmt.Sprintf("%d", skipped)),
		)
		return
	}
	fmt.Fprintf(out, "%s all set — %s gif(s), %s skipped\n",
		s.green("🎉"),
		s.bold(fmt.Sprintf("%d", ok)),
		s.dim(fmt.Sprintf("%d", skipped)),
	)
}

func stageEmoji(stage string) string {
	switch stage {
	case "reading":
		return "📂"
	case "encoding":
		return "🎨"
	case "writing":
		return "💾"
	default:
		return "✨"
	}
}

// progressPrinter draws a live progress line on a TTY (stderr), or is a no-op.
type progressPrinter struct {
	w      io.Writer
	s      styleWriter
	mu     sync.Mutex
	active bool
	label  string
}

func newProgress(w io.Writer, label string) *progressPrinter {
	if !isWriterTerminal(w) {
		return &progressPrinter{w: w, active: false}
	}
	return &progressPrinter{
		w:      w,
		s:      newStyle(w),
		active: true,
		label:  label,
	}
}

func (p *progressPrinter) handler() func(convert.Progress) {
	if p == nil || !p.active {
		return nil
	}
	return func(pr convert.Progress) {
		p.render(pr)
	}
}

func (p *progressPrinter) render(pr convert.Progress) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.active {
		return
	}
	pct := pr.Percent
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	bar := progressBar(pct, 24, p.s.color)
	label := p.label
	if label == "" {
		label = "converting"
	}
	line := fmt.Sprintf("\r%s %s %s %s %3d%% (%d/%d)   ",
		stageEmoji(pr.Stage),
		p.s.bold(label),
		bar,
		p.s.dim(pr.Stage),
		pct,
		pr.Done,
		pr.Total,
	)
	fmt.Fprint(p.w, line)
}

func (p *progressPrinter) finish(ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.active {
		return
	}
	// Clear the progress line, then leave a tidy status.
	fmt.Fprint(p.w, "\r"+strings.Repeat(" ", 72)+"\r")
	if ok {
		fmt.Fprintf(p.w, "%s %s\n", p.s.green("✨"), p.s.dim("frames locked in"))
	}
	p.active = false
}

func progressBar(pct, width int, color bool) string {
	if width < 4 {
		width = 4
	}
	filled := pct * width / 100
	if filled > width {
		filled = width
	}
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < width; i++ {
		switch {
		case i < filled:
			if color {
				b.WriteString(ansiCyan)
			}
			b.WriteRune('█')
			if color {
				b.WriteString(ansiReset)
			}
		case i == filled && pct < 100:
			if color {
				b.WriteString(ansiMagenta)
			}
			frames := []rune{'░', '▒', '▓', '█'}
			b.WriteRune(frames[(time.Now().UnixNano()/5e7)%int64(len(frames))])
			if color {
				b.WriteString(ansiReset)
			}
		default:
			b.WriteRune('░')
		}
	}
	b.WriteString("]")
	return b.String()
}

func promptPrefix(out io.Writer, emoji string) string {
	s := newStyle(out)
	if emoji == "" {
		return s.cyan("›") + " "
	}
	return emoji + " "
}

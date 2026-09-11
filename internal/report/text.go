package report

import (
	"fmt"
	"io"
	"os"
	"time"
)

type TextReporter struct{}

func (r *TextReporter) Report(w io.Writer, res Result) error {
	verdict := "MISS"
	if res.Match {
		verdict = "MATCH"
		if useColor(w) {
			verdict = "\x1b[32m" + verdict + "\x1b[0m"
		}
	}
	if _, err := fmt.Fprintf(w, "[%s] %s\n", verdict, res.Entry.IP.String()); err != nil {
		return err
	}

	if res.Match {
		source := res.Hit.Meta.Source + "/" + res.Hit.Meta.Category
		if useColor(w) {
			source = "\x1b[32m" + source + "\x1b[0m"
		}
		if _, err := fmt.Fprintf(w, "  prefix:   %s (%s)\n", res.Hit.Prefix.String(), source); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "  time: %s | request: %s | referer: %s\n",
		formatTime(res.Entry.Timestamp), formatRequest(res), orDash(res.Entry.Referer)); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "  ua:       %s\n", orDash(res.Entry.UA))
	return err
}

func (r *TextReporter) Flush() error {
	return nil
}

func formatTime(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Format(time.RFC3339)
}

func formatRequest(res Result) string {
	e := res.Entry
	if e.Method == "" && e.Path == "" {
		return "-"
	}
	out := e.Method
	if e.Path != "" {
		out += " " + e.Path
	}
	if e.Status != 0 {
		out += fmt.Sprintf(" -> %d", e.Status)
		if e.Bytes > 0 {
			out += fmt.Sprintf(" (%d bytes)", e.Bytes)
		}
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func useColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

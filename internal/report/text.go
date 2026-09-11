package report

import (
	"fmt"
	"io"
)

type TextReporter struct{}

func (r *TextReporter) Report(w io.Writer, res Result) error {
	matchStr := "MISS "
	if res.Match {
		matchStr = "MATCH"
	}

	src := "-"
	if res.Match {
		src = res.Hit.Meta.Source
	}

	_, err := fmt.Fprintf(w, "[%s] %s %s\n", matchStr, res.Entry.IP.String(), src)
	return err
}

func (r *TextReporter) Flush() error {
	return nil
}

package report

import (
	"fmt"
	"github.com/saurabhsharma2u/botcheck/internal/logparse"
	"github.com/saurabhsharma2u/botcheck/internal/matcher"
	"io"
)

type Result struct {
	Entry logparse.Entry
	Hit   matcher.Hit
	Match bool
}

type Reporter interface {
	Report(w io.Writer, res Result) error
	Flush() error
}

func GetReporter(format string) (Reporter, error) {
	switch format {
	case "text":
		return &TextReporter{}, nil
	case "json":
		return &JSONReporter{}, nil
	case "csv":
		return &CSVReporter{}, nil
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

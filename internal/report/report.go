package report

import (
	"fmt"
	"io"
	"github.com/saurabhsharma2u/iambot/internal/logparse"
	"github.com/saurabhsharma2u/iambot/internal/matcher"
)

type Result struct {
	Entry logparse.Entry
	Hit   matcher.Hit
	Match bool
}

type Reporter interface {
	Report(w io.Writer, res Result) error
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

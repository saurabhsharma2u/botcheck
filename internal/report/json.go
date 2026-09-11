package report

import (
	"encoding/json"
	"io"
)

type JSONReporter struct{}

func (r *JSONReporter) Report(w io.Writer, res Result) error {
	out := map[string]interface{}{
		"ip":      res.Entry.IP.String(),
		"matched": res.Match,
	}
	if res.Match {
		out["meta"] = res.Hit.Meta
	}

	b, err := json.Marshal(out)
	if err != nil {
		return err
	}

	_, err = w.Write(append(b, '\n'))
	return err
}

func (r *JSONReporter) Flush() error {
	return nil
}

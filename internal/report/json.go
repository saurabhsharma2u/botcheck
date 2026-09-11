package report

import (
	"encoding/json"
	"io"
)

type JSONReporter struct{}

func (r *JSONReporter) Report(w io.Writer, res Result) error {
	e := res.Entry
	out := map[string]interface{}{
		"ip":      e.IP.String(),
		"method":  e.Method,
		"path":    e.Path,
		"status":  e.Status,
		"bytes":   e.Bytes,
		"ua":      e.UA,
		"referer": e.Referer,
		"matched": res.Match,
	}
	if !e.Timestamp.IsZero() {
		out["timestamp"] = e.Timestamp.Format("2006-01-02T15:04:05Z07:00")
	}
	if res.Match {
		out["prefix"] = res.Hit.Prefix.String()
		out["source"] = res.Hit.Meta.Source
		out["category"] = res.Hit.Meta.Category
		out["name"] = res.Hit.Meta.Name
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

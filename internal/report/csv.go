package report

import (
	"encoding/csv"
	"io"
	"strconv"
)

type CSVReporter struct {
	w       *csv.Writer
	written bool
}

func (r *CSVReporter) Report(w io.Writer, res Result) error {
	if r.w == nil {
		r.w = csv.NewWriter(w)
	}

	if !r.written {
		if err := r.w.Write([]string{"timestamp", "ip", "matched", "prefix", "source", "category", "method", "path", "status", "bytes", "ua", "referer"}); err != nil {
			return err
		}
		r.written = true
	}

	e := res.Entry
	timestamp := ""
	if !e.Timestamp.IsZero() {
		timestamp = e.Timestamp.Format("2006-01-02T15:04:05Z07:00")
	}
	prefix, source, category := "", "", ""
	if res.Match {
		prefix = res.Hit.Prefix.String()
		source = res.Hit.Meta.Source
		category = res.Hit.Meta.Category
	}
	record := []string{
		timestamp,
		e.IP.String(),
		strconv.FormatBool(res.Match),
		prefix,
		source,
		category,
		e.Method,
		e.Path,
		nonZero(e.Status),
		nonZero64(e.Bytes),
		e.UA,
		e.Referer,
	}

	return r.w.Write(record)
}

func (r *CSVReporter) Flush() error {
	if r.w == nil {
		return nil
	}
	r.w.Flush()
	return r.w.Error()
}

func nonZero(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}

func nonZero64(v int64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(v, 10)
}

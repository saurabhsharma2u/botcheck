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
		if err := r.w.Write([]string{"IP", "Matched", "Source", "Category"}); err != nil {
			return err
		}
		r.written = true
	}

	src := ""
	cat := ""
	if res.Match {
		src = res.Hit.Meta.Source
		cat = res.Hit.Meta.Category
	}

	record := []string{
		res.Entry.IP.String(),
		strconv.FormatBool(res.Match),
		src,
		cat,
	}

	if err := r.w.Write(record); err != nil {
		return err
	}
	return nil
}

func (r *CSVReporter) Flush() error {
	if r.w == nil {
		return nil
	}
	r.w.Flush()
	return r.w.Error()
}

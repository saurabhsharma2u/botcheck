package logparse

import "testing"

var benchCommonLine = `205.234.147.102 - - [11/Sep/2026:00:08:19 +0000] "GET /country/india HTTP/2.0" 200 30991 "-" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"`

var benchJSONLine = `{"ip":"205.234.147.102","status":200,"method":"GET","path":"/country/india"}`

func BenchmarkCommonParser(b *testing.B) {
	p := NewCommonParser()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Parse(benchCommonLine); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONParser(b *testing.B) {
	p := NewJSONParser()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Parse(benchJSONLine); err != nil {
			b.Fatal(err)
		}
	}
}

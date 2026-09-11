package logparse

type cloudflareParser struct {
	jsonParser *jsonParser
}

func NewCloudflareParser() LogParser {
	return &cloudflareParser{jsonParser: &jsonParser{}}
}

func (p *cloudflareParser) Name() string {
	return "cloudflare"
}

func (p *cloudflareParser) Parse(line string) (Entry, error) {
	// Often Cloudflare logs are sent as JSON.
	return p.jsonParser.Parse(line)
}

package concept

import "testing"

func TestR8e3CanonicalNameSuggestions(t *testing.T) {
	for _, test := range []struct {
		name  string
		style string
		want  string
	}{
		{"process_market_data", "PascalCase", "ProcessMarketData"},
		{"parse_http_request", "PascalCase", "ParseHttpRequest"},
		{"gpu_buffer", "PascalCase", "GpuBuffer"},
		{"xml_document", "PascalCase", "XmlDocument"},
		{"packet_count", "camelCase", "packetCount"},
		{"http_status", "camelCase", "httpStatus"},
		{"GPU_buffer", "camelCase", "gpuBuffer"},
	} {
		if got := declarationNameSuggestion(test.name, test.style); got != test.want {
			t.Errorf("%s as %s: got %q, want %q", test.name, test.style, got, test.want)
		}
	}
}

package antivirus

import "testing"

func TestParseClamdResponse(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Status
	}{
		{name: "clean", line: "stream: OK\n", want: StatusClean},
		{name: "infected", line: "stream: Eicar-Test-Signature FOUND\n", want: StatusInfected},
		{name: "too large", line: "INSTREAM size limit exceeded. ERROR\n", want: StatusTooLarge},
		{name: "unknown", line: "wat\n", want: StatusUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseClamdResponse(tt.line, 0)
			if got.Status != tt.want {
				t.Fatalf("status = %s, want %s", got.Status, tt.want)
			}
		})
	}
}

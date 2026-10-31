package config

import "testing"

func TestValidateICEServers(t *testing.T) {
	const goodSecret = "a-sufficiently-long-secret"
	goodTURN := []string{"turn:turn.example.com:3478?transport=udp"}
	goodSTUN := []string{"stun:stun.l.google.com:19302"}

	tests := []struct {
		name    string
		secret  string
		turn    []string
		stun    []string
		wantErr bool
	}{
		{"stun only, no turn", "", nil, goodSTUN, false},
		{"valid turn + secret + stun", goodSecret, goodTURN, goodSTUN, false},
		{"turn url with wrong scheme", goodSecret, []string{"https://turn.example.com"}, goodSTUN, true},
		{"stun url with turn scheme", goodSecret, goodTURN, []string{"turn:nope:3478"}, true},
		{"turn set but secret too short", "short", goodTURN, goodSTUN, true},
		{"turn set but secret empty", "", goodTURN, goodSTUN, true},
		{"secret without turn urls is harmless (stun-only)", goodSecret, nil, goodSTUN, false},
		{"empty everything is fine", "", nil, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateICEServers(tt.secret, tt.turn, tt.stun)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateICEServers() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

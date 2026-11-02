package services

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	testSTUN = []string{"stun:stun.l.google.com:19302"}
	testTURN = []string{"turn:turn.test:3478?transport=udp", "turn:turn.test:3478?transport=tcp"}
)

func TestICEServers(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		turnURLs  []string
		wantTURN  bool
		wantCount int
	}{
		{"stun only when secret empty", "", testTURN, false, 1},
		{"stun only when no turn urls", "topsecret", nil, false, 1},
		{"stun plus turn when configured", "topsecret", testTURN, true, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTURNService(tt.secret, tt.turnURLs, testSTUN, time.Hour)
			servers := svc.ICEServers("user-123")

			if len(servers) != tt.wantCount {
				t.Fatalf("got %d ice servers, want %d", len(servers), tt.wantCount)
			}

			// First entry is always STUN — no credentials.
			if servers[0].Username != "" || servers[0].Credential != "" {
				t.Errorf("STUN entry must carry no credentials, got user=%q cred=%q", servers[0].Username, servers[0].Credential)
			}

			if !tt.wantTURN {
				return
			}

			// All TURN entries (servers[1:]) must carry the same minted credential.
			first := servers[1]
			if first.Username == "" || first.Credential == "" {
				t.Fatalf("TURN entry missing credentials: %+v", first)
			}
			for i, turn := range servers[1:] {
				if turn.Username != first.Username || turn.Credential != first.Credential {
					t.Errorf("TURN entry %d credential differs: got user=%q cred=%q, want user=%q cred=%q",
						i+1, turn.Username, turn.Credential, first.Username, first.Credential)
				}
				if len(turn.URLs) != 1 || turn.URLs[0] != tt.turnURLs[i] {
					t.Errorf("TURN entry %d url = %v, want %q", i+1, turn.URLs, tt.turnURLs[i])
				}
			}
		})
	}
}

func TestMintCredentialFormat(t *testing.T) {
	const secret = "topsecret"
	const userID = "user-123"
	ttl := time.Hour

	svc := NewTURNService(secret, testTURN, testSTUN, ttl).(*turnService)
	username, credential := svc.mintCredential(userID)

	// username = <unix_exp>:<userID>
	parts := strings.SplitN(username, ":", 2)
	if len(parts) != 2 || parts[1] != userID {
		t.Fatalf("username %q is not <exp>:<userID>", username)
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		t.Fatalf("expiry not an int: %v", err)
	}
	if delta := exp - time.Now().Unix(); delta < int64(ttl.Seconds())-5 || delta > int64(ttl.Seconds())+5 {
		t.Errorf("expiry %d not ~ttl from now (delta=%ds)", exp, delta)
	}

	// credential = base64(HMAC-SHA1(secret, username))
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(username))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if credential != want {
		t.Errorf("credential mismatch: got %q want %q", credential, want)
	}
}

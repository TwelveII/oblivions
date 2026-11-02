package services

import (
	"crypto/hmac"
	"crypto/sha1" // HMAC-SHA1 is mandated by the coturn TURN REST API (use-auth-secret); not a hashing security choice.
	"encoding/base64"
	"fmt"
	"time"

	"github.com/akinalp/mqvi/models"
)

// ICEServerProvider builds the WebRTC ICE server list (STUN + TURN) for a user,
// minting a fresh short-lived HMAC TURN credential on each call. P2P 1-on-1
// calls are central and friendship-scoped, so one provider serves every call.
type ICEServerProvider interface {
	ICEServers(userID string) []models.ICEServer
}

type turnService struct {
	secret   string
	turnURLs []string
	stunURLs []string
	ttl      time.Duration
}

// NewTURNService constructs an ICEServerProvider. When secret or turnURLs is
// empty it returns STUN-only — P2P still connects on non-restrictive networks,
// it just loses the relay fallback (graceful degrade, no error).
//
// ttl is the validated credential lifetime; config.Load is the single source of
// truth for it (positive, bounded), so this constructor trusts the value.
func NewTURNService(secret string, turnURLs, stunURLs []string, ttl time.Duration) ICEServerProvider {
	return &turnService{
		secret:   secret,
		turnURLs: turnURLs,
		stunURLs: stunURLs,
		ttl:      ttl,
	}
}

func (s *turnService) ICEServers(userID string) []models.ICEServer {
	servers := make([]models.ICEServer, 0, len(s.stunURLs)+len(s.turnURLs))

	for _, url := range s.stunURLs {
		servers = append(servers, models.ICEServer{URLs: []string{url}})
	}

	if s.secret != "" && len(s.turnURLs) > 0 {
		username, credential := s.mintCredential(userID)
		for _, url := range s.turnURLs {
			servers = append(servers, models.ICEServer{
				URLs:       []string{url},
				Username:   username,
				Credential: credential,
			})
		}

		// FUTURE (fleet expansion): append coturn endpoints running on
		// platform-managed LiveKit hosts here — query livekit_instances where
		// is_platform_managed = 1 and emit one ICE server per host. The same
		// shared secret authenticates all of them, so reuse the credential above.
		// Keep coturn relay ports below every LiveKit rtc.port_range (50000-60000
		// across hosts) so the two never collide on a co-located node.
		//
		// FUTURE (load-aware routing): instead of returning every TURN endpoint,
		// pick by least bandwidth in use across the fleet (TURN relay headroom,
		// NOT SFU participant count). Until then, handing the client multiple
		// iceServers lets ICE select a working relay on its own.
	}

	return servers
}

// mintCredential builds a coturn use-auth-secret credential:
//
//	username   = <unix_expiry>:<userID>
//	credential = base64(HMAC-SHA1(secret, username))
func (s *turnService) mintCredential(userID string) (username, credential string) {
	exp := time.Now().Add(s.ttl).Unix()
	username = fmt.Sprintf("%d:%s", exp, userID)
	mac := hmac.New(sha1.New, []byte(s.secret))
	mac.Write([]byte(username))
	credential = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return username, credential
}

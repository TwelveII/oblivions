package models

// ICEServer mirrors the WebRTC RTCIceServer shape returned to P2P call clients.
// STUN entries omit credentials; TURN entries carry a short-lived HMAC
// username/credential.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// ICEServersResponse is the payload of GET /api/calls/ice-servers.
type ICEServersResponse struct {
	ICEServers []ICEServer `json:"ice_servers"`
}

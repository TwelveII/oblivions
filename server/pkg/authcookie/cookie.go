package authcookie

import (
	"net/http"
	"time"
)

const Name = "mqvi_file_session"

const Path = "/api/files"

func Set(w http.ResponseWriter, fileToken string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     Name,
		Value:    fileToken,
		Path:     Path,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})
}

func Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     Name,
		Value:    "",
		Path:     Path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})
}

func Read(r *http.Request) string {
	c, err := r.Cookie(Name)
	if err != nil {
		return ""
	}
	return c.Value
}

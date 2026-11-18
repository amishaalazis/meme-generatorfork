// Package cookie berisi helper cookie auth.
package cookie

import "net/http"

const (
	AccessTokenName  = "meme_access_token"
	RefreshTokenName = "meme_refresh_token"
)

// Set menyimpan cookie dengan HttpOnly=true dan Path="/" agar setiap
// permintaan dari browser secara otomatis membawa cookie tersebut.
// HttpOnly=true mencegah JavaScript (termasuk XSS) membaca token langsung.
func Set(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear menghapus cookie dengan atribut Path dan HttpOnly yang identik
// dengan Set agar browser benar-benar menghapus cookie tersebut.
func Clear(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

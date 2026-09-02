package api

import (
	"net/http"
	"strconv"

	log "github.com/gophish/gophish/logger"
	qrcode "github.com/skip2/go-qrcode"
)

// QRCode generates a QR code PNG for an arbitrary URL or text. It backs the
// standalone QR generator, used for physical quishing tests (posters, printed
// codes, etc.). Query params: text (required), size (pixels, 64-2048).
func (as *Server) QRCode(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	if text == "" {
		http.Error(w, "Missing 'text' parameter", http.StatusBadRequest)
		return
	}
	size := 512
	if s := r.URL.Query().Get("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 64 && n <= 2048 {
			size = n
		}
	}
	png, err := qrcode.Encode(text, qrcode.Medium, size)
	if err != nil {
		log.Error(err)
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(png)
}

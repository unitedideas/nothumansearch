package handlers

import "net/http"

// protectReceiptBearingResponse prevents browser, proxy, and referer leakage
// of per-search bearer capabilities. Public organic content remains freely
// accessible; only responses carrying or consuming a search receipt are
// deliberately non-cacheable.
func protectReceiptBearingResponse(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

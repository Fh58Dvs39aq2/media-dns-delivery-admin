package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	dnsdelivery "example.com/media-dns-admin"
)

type server struct {
	dns  *dnsdelivery.Client
	page *template.Template
}

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	page := template.Must(template.New("admin").Parse(adminHTML))
	s := &server{dns: dnsdelivery.NewClient(apiKey), page: page}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.admin)
	mux.HandleFunc("POST /api/deliveries", s.publish)
	mux.HandleFunc("GET /api/records", s.list)
	mux.HandleFunc("DELETE /api/records", s.delete)

	addr := envOr("ADDR", ":8080")
	log.Printf("media DNS admin listening on %s", addr)
	httpServer := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(httpServer.ListenAndServe())
}

func (s *server) admin(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.page.Execute(w, nil); err != nil {
		log.Printf("render admin: %v", err)
	}
}

func (s *server) publish(w http.ResponseWriter, r *http.Request) {
	var asset dnsdelivery.AssetDelivery
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	record, err := s.dns.Publish(r.Context(), asset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	records, err := s.dns.List(r.Context(), r.URL.Query().Get("domain"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *server) delete(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if err := s.dns.Delete(r.Context(), query.Get("domain"), query.Get("record_id"), query.Get("record_type"), query.Get("name")); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	var apiErr *dnsdelivery.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		status = apiErr.StatusCode
	} else if errors.Is(err, dnsdelivery.ErrProcessing) {
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

const adminHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<title>Media DNS delivery</title>
<style>body{font:15px system-ui;max-width:680px;margin:3rem auto;padding:0 1rem;color:#17202a}form{display:grid;gap:.8rem}label{display:grid;gap:.25rem}input,select,button{font:inherit;padding:.65rem;border:1px solid #9aa4ad;border-radius:4px}button{background:#176b4d;color:white;border:0;cursor:pointer}pre{background:#f3f5f6;padding:1rem;white-space:pre-wrap}</style></head>
<body><h1>Media DNS delivery</h1><form id="delivery">
<label>Asset ID<input name="asset_id" required></label>
<label>Domain<input name="domain" placeholder="media.example.com" required></label>
<label>Origin host<input name="origin_host" placeholder="edge.example.net" required></label>
<label>Processing job<select name="job_status"><option value="processing">processing</option><option value="ready">ready</option></select></label>
<button>Publish creator hostname</button></form><pre id="result">Waiting for an asset.</pre>
<script>document.querySelector('#delivery').addEventListener('submit',async(e)=>{e.preventDefault();const body=Object.fromEntries(new FormData(e.target));const res=await fetch('/api/deliveries',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});document.querySelector('#result').textContent=JSON.stringify(await res.json(),null,2)});</script></body></html>`

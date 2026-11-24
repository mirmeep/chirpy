package main

import (
	"fmt"
	"net/http"
	"time"
	"log"
	"sync/atomic"
	"encoding/json"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("<html>"+
			"<body>" +
				"<h1>Welcome, Chirpy Admin</h1>" +
				"<p>Chirpy has been visited %d times!</p>" +
			"</body>" +
		"</html>", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Hits", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerValidate(w http.ResponseWriter, r *http.Request) {
	type message struct{
		Body string `json:"body"`
	}
	
	decoder := json.NewDecoder(r.Body)
	msg := message{}
	err := decoder.Decode(&msg)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	bodyLength := len(msg.Body)

	if bodyLength > 140 {
		respondWithError(w, 400, "Chirp is too long")
	} else {
		respondWithJSON(w, 200)
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnErr struct {
		Msg string `json:"error"`
	}

	respBody := returnErr{
		Msg: msg,
	}

	data, err := json.Marshal(respBody) 
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithJSON(w http.ResponseWriter, code int) {
	type returnVal struct {
		IsValid bool `json:"valid"`
	}

	respBody := returnVal{
		IsValid: true,
	}

	data, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func main() {
	var apiCfg apiConfig
	mux := http.NewServeMux()
	handler := http.FileServer(http.Dir("."))

	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(handler)))
	
	mux.HandleFunc("GET  /admin/metrics", 		apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset",   		apiCfg.handlerReset)
	mux.HandleFunc("GET  /api/healthz", 		apiCfg.handlerHealthz)
	mux.HandleFunc("POST /api/validate_chirp", 	apiCfg.handlerValidate)

	srv := http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		// Use CrossOriginProtection.Handler to block all non-safe cross-origin
		// browser requests to mux.
		Handler: http.NewCrossOriginProtection().Handler(mux),
	}

	log.Fatal(srv.ListenAndServe())
}
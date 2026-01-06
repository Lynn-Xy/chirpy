package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"log"
	"os"
	"database/sql"
	"strings"
	_ "github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/Lynn-Xy/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
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
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>\n`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func handlerValidandProfane(w http.ResponseWriter, r *http.Request) {
	type returnError struct {
		Error string `json:"error"`	
	}
	type returnBody struct {
		Body string `json:"body"`
	}
	type cleanedBody struct {
		Cleaned_body string `json:"cleaned_body"`
	}
	var reqBody returnBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Printf("Error decoding JSON: %s", err)
		response := returnError{Error: "invalid JSON"}
		data, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			w.Write(data)
			return
		}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(400)
	w.Write(data)
	return
	}
	if len(reqBody.Body) > 140 {
		response := returnError{Error: "Chirp is too long"}
		data, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			w.Write(data)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(400)
		w.Write(data)
		return
	}
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	cleaned := reqBody.Body
	words := strings.Fields(reqBody.Body)
	for i, word := range words {
		lowered := strings.ToLower(word)
		for _, profane := range profaneWords {
			if lowered == profane {
				words[i] = strings.Repeat("*", 4)
			}
		}
	}
	cleaned = strings.Join(words, " ")
	response := cleanedBody{Cleaned_body: cleaned}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		w.Write(data)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Write(data)
}

func main() {
	err1 := godotenv.Load()
	DB_URL := os.Getenv("DB_URL")
	if err1 != nil {
		log.Printf("Error loading .env file: %v", err1)
	}
	db, err2 := sql.Open("postgres", DB_URL)
	if err2 != nil {
		log.Fatalf("Failed to connect to database: %v", err2)
	}
	defer db.Close()
	dbQuery := database.New(db)
	cfg := apiConfig{dbQueries: dbQuery}
	newServerMux := http.NewServeMux()
	newServer := &http.Server{
		Addr:    ":8080",
		Handler: newServerMux,
	}
	newServerMux.Handle("/app/", http.StripPrefix("/app/", cfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	newServerMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})
	newServerMux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	newServerMux.HandleFunc("POST /admin/reset", cfg.handlerReset)
	newServerMux.HandleFunc("POST /api/validate_chirp", handlerValidandProfane)
	fmt.Println("Starting server on :8080")
	err3 := newServer.ListenAndServe()
	if err3 != nil {
		fmt.Printf("server failed to start: %v", err3)
	}
}
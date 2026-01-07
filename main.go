package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"log"
	"context"
	"os"
	"database/sql"
	"strings"
	"time"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"github.com/Lynn-Xy/chirpy/internal/database"
	"github.com/Lynn-Xy/chirpy/internal/auth"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	Platform string
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

func (cfg *apiConfig) handlerDeleteAllUsers(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "Dev" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
		return
	}
	err := cfg.dbQueries.DeleteAllUsers(context.Background())
	if err != nil {
		log.Printf("Error deleting all users: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	type responseBody struct {
		Id uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email string `json:"email"`
	}
	var reqBody requestBody
	err1 := json.NewDecoder(r.Body).Decode(&reqBody)
	if err1 != nil {
		log.Printf("Error decoding JSON: %s", err1)
		w.WriteHeader(400)
		return
	}
	hashedPassword, err3 := auth.HashPassword(reqBody.Password)
	if err3 != nil {
		log.Printf("Error hashing password: %s", err3)
		w.WriteHeader(500)
		return
	}
	newUser, err2 := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email: reqBody.Email,
		HashedPassword: hashedPassword,
	})
	if err2 != nil {
		log.Printf("Error creating user in database: %s", err2)
		w.WriteHeader(500)
		return
	}
	response := responseBody{
		Id: newUser.ID,
		Created_at: newUser.CreatedAt,
		Updated_at: newUser.UpdatedAt,
		Email: newUser.Email,
	}
	data, err3 := json.Marshal(response)
	if err3 != nil {
		log.Printf("Error marshalling JSON: %s", err3)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(201)
	w.Write(data)
}

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	type responseUser struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
	}
	
	var reqBody requestBody
	err1 := json.NewDecoder(r.Body).Decode(&reqBody)
	if err1 != nil {
		log.Printf("Error decoding JSON: %s", err1)
		w.WriteHeader(400)
		return
	}

	user, err2 := cfg.dbQueries.GetUserByEmail(r.Context(), reqBody.Email)
	if err2 != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Incorrect email or password"}`))
		return
	}
	match, err3 := auth.CheckPasswordHash(reqBody.Password, user.HashedPassword)
	if err3 != nil || !match {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Incorrect email or password"}`))
		return
	}
	response := responseUser{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	data, err4 := json.Marshal(response)
	if err4 != nil {
		log.Printf("Error marshalling JSON: %s", err4)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	type chirpResponse struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	chirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		log.Printf("Error getting all chirps: %s", err)
		w.WriteHeader(500)
		return
	}
	response := make([]chirpResponse, 0, len(chirps))
	for _, chirp := range chirps {
		response = append(response, chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	type chirpResponse struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		log.Printf("Error parsing chirp ID: %s", err)
		w.WriteHeader(400)
		return
	}
	chirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		log.Printf("Error getting chirp: %s", err)
		w.WriteHeader(404)
		return
	}
	response := chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *apiConfig) handlerPublishChirp(w http.ResponseWriter, r *http.Request) {
	type returnError struct {
		Error string `json:"error"`	
	}
	type requestBody struct {
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	var reqBody requestBody
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
	type responseChirp struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	
	chirp, err := cfg.dbQueries.PublishChirp(r.Context(), database.PublishChirpParams{
		Body:   cleaned,
		UserID: reqBody.UserID,
	})
	if err != nil {
		log.Printf("Error publishing chirp to database: %s", err)
		w.WriteHeader(500)
		return
	}

	response := responseChirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      cleaned,
		UserID:    reqBody.UserID,
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		w.Write(data)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(201)
	w.Write(data)
}

func main() {
	err1 := godotenv.Load()
	if err1 != nil {
		log.Printf("Error loading .env file: %v", err1)
	}
	DB_URL := os.Getenv("DB_URL")
	Plat := os.Getenv("PLATFORM")
	db, err2 := sql.Open("postgres", DB_URL)
	if err2 != nil {
		log.Fatalf("Failed to connect to database: %v", err2)
	}
	defer db.Close()
	dbQuery := database.New(db)
	cfg := apiConfig{
		dbQueries: dbQuery,
		Platform:  Plat,
	}
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
	newServerMux.HandleFunc("POST /admin/reset", cfg.handlerDeleteAllUsers)
	newServerMux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	newServerMux.HandleFunc("POST /api/login", cfg.handlerUserLogin)
	newServerMux.HandleFunc("POST /api/chirps", cfg.handlerPublishChirp)
	newServerMux.HandleFunc("GET /api/chirps", cfg.handlerGetAllChirps)
	newServerMux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handlerGetChirp)
	fmt.Println("Starting server on :8080")
	err3 := newServer.ListenAndServe()
	if err3 != nil {
		fmt.Printf("server failed to start: %v", err3)
	}
}
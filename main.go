package main
godotenv.Load()
dbURL := os.Getenv("DB_URL")
db, err := sql.Open("postgres", dbURL)


import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func main() {
	serveMux := http.NewServeMux()
	apiConfig := &apiConfig{}
	apiConfig.fileServerHits.Store(0)
	handler := http.StripPrefix("/app", http.FileServer(http.Dir("./")))
	serveMux.Handle("/app/", apiConfig.middlewareMetricsInc(handler))
	serveMux.HandleFunc("GET /admin/metrics", apiConfig.handleNumberOfHits)
	serveMux.HandleFunc("POST /admin/reset", apiConfig.handleResetHits)
	serveMux.HandleFunc("POST /api/validate_chirp", apiConfig.handleApiValidity)

	serveMux.HandleFunc("GET /api/healthz", handleHttpReadiness)

	server := &http.Server {
		Addr: ":8080",
		Handler: serveMux,
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error occurred: ", err)
	}
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		fmt.Println("Request received, hits incremented")
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handleNumberOfHits(w http.ResponseWriter, r *http.Request) {
	r.Header.Add("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf(
	`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load())))
}

func (cfg *apiConfig) handleApiValidity(w http.ResponseWriter, r *http.Request) {
	type reqBody struct {
		Body string `json:"body"`
	}

	type resBody struct {
		Cleaned_body string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	body := &reqBody{}
	err := decoder.Decode(&body)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
  }

	if len(body.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	bad_words := map[string]struct{}{
		"kerfuffle": {},
		"sharbert": {},
		"fornax": {},
	}

	respondWithJSON(w, http.StatusOK, &resBody{Cleaned_body: replaceBadWords(body.Body, bad_words)})
}

func (cfg *apiConfig) handleResetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileServerHits.Store(0)
}

func handleHttpReadiness(w http.ResponseWriter, r *http.Request) {
	r.Header.Add("Content-Type", "text/plain; charset=utf-8")

	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type resError struct {
		Error string `json:"error"`
	}

	w.WriteHeader(code)
	resp, _ := json.Marshal(resError{Error: msg})
	w.Write(resp)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.WriteHeader(code)
	resp, _ := json.Marshal(payload)
	w.Write(resp)
}

func replaceBadWords(body string, badWords map[string]struct{}) string {
	 words := strings.Split(body, " ")

	for i, word := range words {
		loweredWord := strings.ToLower(word)
		if _, ok := badWords[loweredWord]; ok {
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}

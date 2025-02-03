package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type PingResult struct {
	IP          string  `json:"ip"`
	PingTime    float64 `json:"ping_time"`
	LastSuccess string  `json:"last_success"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Created     string  `json:"created"`
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ping_results (
			ip VARCHAR(50),
			ping_time FLOAT,
			last_success TIMESTAMP,
			name VARCHAR(100),
			status VARCHAR(100),
			created TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (ip)
		)
	`)
	return err
}

func main() {
	connStr := "postgres://user:password@db:5432/pingdb?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()

	// Добавляем CORS
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
	)

	r.HandleFunc("/api/ping-results", func(w http.ResponseWriter, r *http.Request) {
		results := []PingResult{}
		rows, err := db.Query(`
			SELECT ip, ping_time, last_success, name, status, created 
			FROM ping_results 
			ORDER BY updated_at DESC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var result PingResult
			err := rows.Scan(
				&result.IP,
				&result.PingTime,
				&result.LastSuccess,
				&result.Name,
				&result.Status,
				&result.Created,
			)
			if err != nil {
				continue
			}
			results = append(results, result)
		}

		json.NewEncoder(w).Encode(results)
	}).Methods("GET")

	r.HandleFunc("/api/ping-results", func(w http.ResponseWriter, r *http.Request) {
		var result PingResult
		if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err := db.Exec(`
			INSERT INTO ping_results (ip, ping_time, last_success, name, status, created, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
			ON CONFLICT (ip) 
			DO UPDATE SET 
				ping_time = $2,
				last_success = $3,
				status = $5,
				updated_at = CURRENT_TIMESTAMP
		`, result.IP, result.PingTime, result.LastSuccess, result.Name, result.Status, result.Created)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}).Methods("POST")

	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(r)))
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
}

type PingResult struct {
	IP          string  `json:"ip"`
	PingTime    float64 `json:"ping_time"`
	LastSuccess string  `json:"last_success"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Created     string  `json:"created"`
}

var jwtKey = []byte("your_secret_key")

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func generateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func validateJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
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

func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("Authorization")
		if tokenStr == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		claims, err := validateJWT(tokenStr)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "username", claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Здесь добавьте проверку учетных данных
	if creds.Username != "admin" || creds.Password != "password" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(creds.Username)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   token,
		Expires: time.Now().Add(5 * time.Minute),
	})
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	// Настройка логирования
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

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

	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/api/ping-results", func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.WithLabelValues("/api/ping-results").Inc()
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

	r.HandleFunc("/login", login).Methods("POST")

	r.Use(jwtMiddleware)

	log.Info("Starting backend service on port 8080")
	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(r)))
}

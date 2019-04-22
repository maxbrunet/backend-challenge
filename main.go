// A simple Web Chat API
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type key int

const (
	requestIDKey key = 0
)

var (
	db      *sqlx.DB
	logger  *log.Logger
	healthy int32
)

// Message represents a posted message
type Message struct {
	Sender         string `db:"sender"          json:"sender"`
	ConversationID int    `db:"conversation_id" json:"conversation_id,omitempty"`
	Message        string `db:"message"         json:"message"`
	Created        string `db:"created"         json:"created,omitempty"`
}

// Conversation represents a conversation with its messages
type Conversation struct {
	ID       int       `json:"id"`
	Messages []Message `json:"messages"`
}

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "server listen address")
	flag.Parse()

	logger = log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	handler := initHTTPHandler()
	db = initDB()

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// go routine waiting on quit channel to gracefully execute Shutdown()
	go func() {
		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	// Wait on shutdown routine to finish
	<-done
	logger.Println("Server stopped")
}

// initHTTPHandler initialize the router
// and return the concatenation of all handlers
func initHTTPHandler() http.Handler {
	nextRequestID := func() string {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/messages/", messagesHandler)
	mux.HandleFunc("/conversations/", conversationsHandler)
	mux.HandleFunc("/healthz", healthzHandler)

	return tracing(nextRequestID)(logging(logger)(mux))
}

// initDB initialize DB connection and schema
func initDB() *sqlx.DB {
	db, err := sqlx.Connect("postgres", os.Getenv("DB_CONNSTR"))
	if err != nil {
		log.Fatalln(err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			sender VARCHAR(32) NOT NULL,
			conversation_id INTEGER NOT NULL,
			message TEXT NOT NULL,
			created TIMESTAMP DEFAULT(CURRENT_TIMESTAMP)
		)`

	db.MustExec(schema)

	return db
}

// messagesHandler handles posting a new message
func messagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(
			w, http.StatusMethodNotAllowed,
			"Only POST is allowed", nil,
		)
		return
	}

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON body", err)
		return
	}
	defer r.Body.Close()

	if msg.ConversationID < 1 {
		respondWithError(
			w, http.StatusBadRequest,
			"Invalid conversation ID", nil,
		)
		return
	}

	insertQuery := `
		INSERT INTO messages (
			sender,
			conversation_id,
			message
		) VALUES (
			:sender,
			:conversation_id,
			:message
		)`

	if _, err := db.NamedExec(insertQuery, msg); err != nil {
		respondWithError(
			w, http.StatusInternalServerError,
			"Failed to post message", err,
		)
		return
	}

	respondWithJSON(
		w, http.StatusOK,
		map[string]string{"message": "Message posted"},
	)
}

// conversationsHandler handles retrieving a conversation by ID
func conversationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(
			w, http.StatusMethodNotAllowed,
			"Only GET is allowed", nil,
		)
		return
	}

	// Extracting ID right after second slash,
	// everything after a third slash is ignored
	ConvID, err := strconv.Atoi(strings.Split(r.RequestURI, "/")[2])
	if err != nil || ConvID < 1 {
		respondWithError(w, http.StatusNotFound, "Invalid conversation ID", err)
		return
	}

	conv := Conversation{
		ID:       ConvID,
		Messages: make([]Message, 0),
	}
	selectQuery := `
		SELECT sender,message,created
		FROM messages
		WHERE conversation_id=$1`

	if err := db.Select(&conv.Messages, selectQuery, conv.ID); err != nil {
		respondWithError(
			w, http.StatusInternalServerError,
			"Failed to retrieve conversation", err,
		)
		return
	}

	respondWithJSON(w, http.StatusOK, conv)
}

// healthzHandler handles HTTP health-check
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&healthy) == 1 {
		respondWithJSON(
			w, http.StatusOK,
			map[string]string{"status": "OK"},
		)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	respondWithError(
		w, http.StatusServiceUnavailable,
		"Unhealthy", nil,
	)
}

// respondWithError write HTTP response with an error message and logs it
func respondWithError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		logger.Printf("%s: %v", message, err)
	} else {
		logger.Printf("%s", message)
	}
	respondWithJSON(w, status, map[string]string{"error": message})
}

// respondWithJSON writes HTTP headers and HTTP response with JSON body
func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	w.Write(response)
}

// logging creates the logging handler and call the next handler
func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(
					requestID,
					r.RemoteAddr,
					r.Method,
					r.URL.Path,
					r.Proto,
					r.UserAgent(),
				)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// tracing creates the tracing handler which adds a request ID and call the next handler
func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

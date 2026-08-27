// Command server runs the Date Chooser HTTP server.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/cfiguerola/date-chooser/internal/store"
	"github.com/cfiguerola/date-chooser/internal/web"
)

const (
	defaultPort   = "8080"
	defaultDBPath = "/data/datechooser.db"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("opening store at %s: %v", dbPath, err)
	}
	defer st.Close()

	srv, err := web.NewServer(st)
	if err != nil {
		log.Fatalf("constructing server: %v", err)
	}

	addr := ":" + port
	log.Printf("date-chooser listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

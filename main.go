// Command catalog-service owns the books and how many copies of each are free.
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dsn := os.Getenv("DB_PATH")
	if dsn == "" {
		dsn = "catalog.db"
	}
	db, err := Open(dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("catalog service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, NewRouter(db)); err != nil {
		log.Fatal(err)
	}
}

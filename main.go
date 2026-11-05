// Command catalog-service owns the books and how many copies are on the shelf.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("catalog service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, NewRouter()))
}

// NewRouter wires the catalog API.
func NewRouter() http.Handler {
	books := newBookStore()

	e := echo.New()
	e.HideBanner = true
	e.Use(RequestIDMiddleware())
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/books/:id", books.getBook)
	e.POST("/books/:id/borrow", books.borrowBook)
	return e
}

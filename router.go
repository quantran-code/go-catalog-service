package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// Book is one title the library holds, with the number of copies on the shelf.
type Book struct {
	ID              int64     `db:"id" json:"id"`
	Title           string    `db:"title" json:"title"`
	Author          string    `db:"author" json:"author"`
	ISBN            string    `db:"isbn" json:"isbn"`
	TotalCopies     int       `db:"total_copies" json:"total_copies"`
	AvailableCopies int       `db:"available_copies" json:"available_copies"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type bookRequest struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	ISBN        string `json:"isbn"`
	TotalCopies int    `json:"total_copies"`
}

func (r bookRequest) validate() string {
	switch {
	case strings.TrimSpace(r.Title) == "":
		return "title is required"
	case strings.TrimSpace(r.Author) == "":
		return "author is required"
	case strings.TrimSpace(r.ISBN) == "":
		return "isbn is required"
	case r.TotalCopies < 1:
		return "total_copies must be at least 1"
	}
	return ""
}

// NewRouter wires the catalog API.
func NewRouter(db *sqlx.DB) http.Handler {
	e := echo.New()
	e.HideBanner = true

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.POST("/books", func(c echo.Context) error {
		var req bookRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		}
		if message := req.validate(); message != "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": message})
		}
		var book Book
		err := db.Get(&book, `INSERT INTO books (title, author, isbn, total_copies, available_copies, created_at)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING *`,
			strings.TrimSpace(req.Title), strings.TrimSpace(req.Author), strings.TrimSpace(req.ISBN),
			req.TotalCopies, req.TotalCopies, time.Now().UTC())
		if err != nil {
			return c.JSON(http.StatusConflict, map[string]string{"error": "isbn already exists"})
		}
		return c.JSON(http.StatusCreated, book)
	})

	e.GET("/books", func(c echo.Context) error {
		page := atoiOr(c.QueryParam("page"), 1)
		size := atoiOr(c.QueryParam("page_size"), 20)
		if page < 1 || size < 1 || size > 100 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid paging"})
		}
		query := strings.TrimSpace(c.QueryParam("q"))
		like := "%" + query + "%"

		var total int
		if err := db.Get(&total, `SELECT count(*) FROM books WHERE (? = '' OR title LIKE ? OR author LIKE ?)`,
			query, like, like); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "counting books"})
		}
		books := []Book{}
		if err := db.Select(&books, `SELECT * FROM books
			WHERE (? = '' OR title LIKE ? OR author LIKE ?)
			ORDER BY id LIMIT ? OFFSET ?`, query, like, like, size, (page-1)*size); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "listing books"})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"items": books, "total": total, "page": page, "page_size": size,
		})
	})

	e.GET("/books/:id", func(c echo.Context) error {
		id, ok := bookID(c)
		if !ok {
			return nil
		}
		book, err := loadBook(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "loading book"})
		}
		return c.JSON(http.StatusOK, book)
	})

	e.PUT("/books/:id", func(c echo.Context) error {
		id, ok := bookID(c)
		if !ok {
			return nil
		}
		var req bookRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		}
		if message := req.validate(); message != "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": message})
		}
		book, err := loadBook(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "loading book"})
		}
		available := book.AvailableCopies + (req.TotalCopies - book.TotalCopies)
		if available < 0 {
			return c.JSON(http.StatusConflict, map[string]string{"error": "total_copies would strand borrowed copies"})
		}
		if _, err := db.Exec(`UPDATE books SET title = ?, author = ?, isbn = ?, total_copies = ?, available_copies = ? WHERE id = ?`,
			strings.TrimSpace(req.Title), strings.TrimSpace(req.Author), strings.TrimSpace(req.ISBN), req.TotalCopies, available, id); err != nil {
			return c.JSON(http.StatusConflict, map[string]string{"error": "isbn already exists"})
		}
		updated, err := loadBook(db, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "loading book"})
		}
		return c.JSON(http.StatusOK, updated)
	})

	e.DELETE("/books/:id", func(c echo.Context) error {
		id, ok := bookID(c)
		if !ok {
			return nil
		}
		result, err := db.Exec(`DELETE FROM books WHERE id = ?`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "deleting book"})
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
		}
		return c.NoContent(http.StatusNoContent)
	})

	// reserve and release are what the loan service calls; they are the only
	// writers of available_copies.
	e.POST("/books/:id/reserve", func(c echo.Context) error {
		return adjustAvailable(c, db, -1)
	})
	e.POST("/books/:id/release", func(c echo.Context) error {
		return adjustAvailable(c, db, +1)
	})

	return e
}

func adjustAvailable(c echo.Context, db *sqlx.DB, delta int) error {
	id, ok := bookID(c)
	if !ok {
		return nil
	}
	book, err := loadBook(db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "book not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "loading book"})
	}
	if delta < 0 && book.AvailableCopies == 0 {
		return c.JSON(http.StatusConflict, map[string]string{"error": "no copies available"})
	}
	if delta > 0 && book.AvailableCopies == book.TotalCopies {
		return c.JSON(http.StatusConflict, map[string]string{"error": "every copy is already on the shelf"})
	}
	if _, err := db.Exec(`UPDATE books SET available_copies = available_copies + ? WHERE id = ?`, delta, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "updating availability"})
	}
	updated, err := loadBook(db, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "loading book"})
	}
	return c.JSON(http.StatusOK, updated)
}

func loadBook(db *sqlx.DB, id int64) (Book, error) {
	var book Book
	err := db.Get(&book, `SELECT * FROM books WHERE id = ?`, id)
	return book, err
}

func bookID(c echo.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		_ = c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid book id"})
		return 0, false
	}
	return id, true
}

func atoiOr(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

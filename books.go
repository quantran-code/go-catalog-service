package main

import (
	"errors"
	"net/http"
	"strconv"
	"sync"

	"github.com/labstack/echo/v4"
)

// BookStatus is whether a book is on the shelf or already out on loan.
// Only two values are defined, and it marshals as a JSON string.
type BookStatus string

const (
	// BookStatusAvailable is the status of a book on the shelf.
	BookStatusAvailable BookStatus = "available"
	// BookStatusOnLoan is the status of a book that is already out.
	BookStatusOnLoan BookStatus = "on_loan"
)

// Book is a title the catalog owns, along with whether it is on the shelf.
type Book struct {
	ID     int64      `json:"id"`
	Title  string     `json:"title"`
	Author string     `json:"author"`
	Status BookStatus `json:"status"`
}

var (
	// errBookNotFound is returned for an id the store does not know.
	errBookNotFound = errors.New("book not found")
	// errBookOnLoan is returned when a borrow finds the book already out.
	errBookOnLoan = errors.New("book already on loan")
)

// errorResponse is the body shape every non-2xx response uses.
type errorResponse struct {
	Error string `json:"error"`
}

// bookStore is the in-memory shelf. mu guards books so that checking a book's
// status and marking it on loan happen as one step.
type bookStore struct {
	mu    sync.Mutex
	books map[int64]Book
}

// newBookStore returns a store seeded with three available books so an unknown
// id is distinguishable from a known one.
func newBookStore() *bookStore {
	return &bookStore{
		books: map[int64]Book{
			1: {ID: 1, Title: "The Go Programming Language", Author: "Donovan and Kernighan", Status: BookStatusAvailable},
			2: {ID: 2, Title: "Designing Data-Intensive Applications", Author: "Kleppmann", Status: BookStatusAvailable},
			3: {ID: 3, Title: "The Pragmatic Programmer", Author: "Hunt and Thomas", Status: BookStatusAvailable},
		},
	}
}

// get returns the book with the given id, or false when the store has no such book.
func (s *bookStore) get(id int64) (Book, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	book, ok := s.books[id]
	return book, ok
}

// borrow checks the book's status and flips it to BookStatusOnLoan inside one
// critical section, so two concurrent borrowers cannot both pass the check.
func (s *bookStore) borrow(id int64) (Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	book, ok := s.books[id]
	if !ok {
		return Book{}, errBookNotFound
	}
	if book.Status == BookStatusOnLoan {
		return Book{}, errBookOnLoan
	}
	book.Status = BookStatusOnLoan
	s.books[id] = book
	return book, nil
}

// getBook serves GET /books/:id. A non-numeric id is treated as a book that does
// not exist rather than as a malformed request.
func (s *bookStore) getBook(c echo.Context) error {
	id, err := parseBookID(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorResponse{Error: errBookNotFound.Error()})
	}

	book, ok := s.get(id)
	if !ok {
		return c.JSON(http.StatusNotFound, errorResponse{Error: errBookNotFound.Error()})
	}
	return c.JSON(http.StatusOK, book)
}

// borrowBook serves POST /books/:id/borrow.
func (s *bookStore) borrowBook(c echo.Context) error {
	id, err := parseBookID(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorResponse{Error: errBookNotFound.Error()})
	}

	book, err := s.borrow(id)
	switch {
	case errors.Is(err, errBookNotFound):
		return c.JSON(http.StatusNotFound, errorResponse{Error: errBookNotFound.Error()})
	case errors.Is(err, errBookOnLoan):
		return c.JSON(http.StatusConflict, errorResponse{Error: errBookOnLoan.Error()})
	case err != nil:
		return c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
	}
	return c.JSON(http.StatusOK, book)
}

// parseBookID reads the :id path parameter.
func parseBookID(c echo.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

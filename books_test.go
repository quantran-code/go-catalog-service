package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func getBook(t *testing.T, router http.Handler, path string) (int, Book) {
	t.Helper()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	var book Book
	if err := json.Unmarshal(rec.Body.Bytes(), &book); err != nil {
		t.Fatalf("decoding %s body %q: %v", path, rec.Body.String(), err)
	}
	return rec.Code, book
}

func borrow(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
	return rec
}

func TestGetBookReturnsTheBook(t *testing.T) {
	code, book := getBook(t, NewRouter(), "/books/1")

	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if book.ID != 1 {
		t.Fatalf("id = %d, want 1", book.ID)
	}
	if book.Title == "" {
		t.Fatal("title is empty")
	}
	if book.Author == "" {
		t.Fatal("author is empty")
	}
	if book.Status != BookStatusAvailable {
		t.Fatalf("status = %q, want %q", book.Status, BookStatusAvailable)
	}
}

func TestGetBookReturns404ForUnknownID(t *testing.T) {
	for _, path := range []string{"/books/999", "/books/abc"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status %d, want 404", rec.Code)
			}
			var body errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decoding body %q: %v", rec.Body.String(), err)
			}
			if body.Error != "book not found" {
				t.Fatalf(`error = %q, want "book not found"`, body.Error)
			}
		})
	}
}

func TestBorrowBookMarksItOnLoan(t *testing.T) {
	router := NewRouter()

	rec := borrow(t, router, "/books/1/borrow")
	if rec.Code != http.StatusOK {
		t.Fatalf("borrow status %d, want 200", rec.Code)
	}
	var borrowed Book
	if err := json.Unmarshal(rec.Body.Bytes(), &borrowed); err != nil {
		t.Fatalf("decoding borrow body %q: %v", rec.Body.String(), err)
	}
	if borrowed.Status != BookStatusOnLoan {
		t.Fatalf("borrow response status = %q, want %q", borrowed.Status, BookStatusOnLoan)
	}

	code, book := getBook(t, router, "/books/1")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if book.Status != BookStatusOnLoan {
		t.Fatalf("status after borrow = %q, want %q", book.Status, BookStatusOnLoan)
	}
}

func TestBorrowBookTwiceReturnsSecondBorrow409(t *testing.T) {
	router := NewRouter()

	if rec := borrow(t, router, "/books/1/borrow"); rec.Code != http.StatusOK {
		t.Fatalf("first borrow status %d, want 200", rec.Code)
	}

	rec := borrow(t, router, "/books/1/borrow")
	if rec.Code != http.StatusConflict {
		t.Fatalf("second borrow status %d, want 409", rec.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body %q: %v", rec.Body.String(), err)
	}
	if body.Error != "book already on loan" {
		t.Fatalf(`error = %q, want "book already on loan"`, body.Error)
	}

	code, book := getBook(t, router, "/books/1")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if book.Status != BookStatusOnLoan {
		t.Fatalf("status = %q, want %q", book.Status, BookStatusOnLoan)
	}
}

func TestBorrowUnknownBookReturns404(t *testing.T) {
	router := NewRouter()

	rec := borrow(t, router, "/books/999/borrow")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body %q: %v", rec.Body.String(), err)
	}
	if body.Error != "book not found" {
		t.Fatalf(`error = %q, want "book not found"`, body.Error)
	}

	// No book is created by a failed borrow.
	if code, _ := getBook(t, router, "/books/999"); code != http.StatusNotFound {
		t.Fatalf("GET /books/999 status %d, want 404", code)
	}
}

func TestConcurrentBorrowsOnlyOneSucceeds(t *testing.T) {
	router := NewRouter()

	const borrowers = 20
	codes := make([]int, borrowers)
	var wg sync.WaitGroup
	for i := 0; i < borrowers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/books/1/borrow", nil))
			codes[i] = rec.Code
		}(i)
	}
	wg.Wait()

	ok, conflict := 0, 0
	for _, code := range codes {
		switch code {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("unexpected status %d", code)
		}
	}
	if ok != 1 {
		t.Fatalf("%d borrows succeeded, want exactly 1", ok)
	}
	if conflict != borrowers-1 {
		t.Fatalf("%d borrows conflicted, want %d", conflict, borrowers-1)
	}
}

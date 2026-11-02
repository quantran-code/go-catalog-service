package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func newTestRouter(t *testing.T) (http.Handler, *sqlx.DB) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return NewRouter(db), db
}

func do(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func createBook(t *testing.T, router http.Handler, title, isbn string, copies int) Book {
	t.Helper()
	rec := do(t, router, http.MethodPost, "/books",
		`{"title":"`+title+`","author":"Author","isbn":"`+isbn+`","total_copies":`+strconv.Itoa(copies)+`}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", title, rec.Code, rec.Body.String())
	}
	var book Book
	if err := json.Unmarshal(rec.Body.Bytes(), &book); err != nil {
		t.Fatal(err)
	}
	return book
}

func TestCreateAndFetchABook(t *testing.T) {
	router, _ := newTestRouter(t)
	book := createBook(t, router, "Dune", "9780441013593", 3)
	if book.AvailableCopies != 3 || book.TotalCopies != 3 {
		t.Fatalf("unexpected availability: %+v", book)
	}
	rec := do(t, router, http.MethodGet, "/books/"+strconv.FormatInt(book.ID, 10), "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Dune") {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}
}

func TestReserveAndReleaseGuardTheShelf(t *testing.T) {
	router, _ := newTestRouter(t)
	book := createBook(t, router, "Dune", "9780441013593", 1)
	path := "/books/" + strconv.FormatInt(book.ID, 10) + "/reserve"

	if rec := do(t, router, http.MethodPost, path, ""); rec.Code != http.StatusOK {
		t.Fatalf("first reserve: %d %s", rec.Code, rec.Body.String())
	}
	rec := do(t, router, http.MethodPost, path, "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "no copies available") {
		t.Fatalf("second reserve: %d %s", rec.Code, rec.Body.String())
	}
	release := "/books/" + strconv.FormatInt(book.ID, 10) + "/release"
	if rec := do(t, router, http.MethodPost, release, ""); rec.Code != http.StatusOK {
		t.Fatalf("release: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, router, http.MethodPost, release, ""); rec.Code != http.StatusConflict {
		t.Fatalf("second release: %d", rec.Code)
	}
}

func TestSearchAndPaging(t *testing.T) {
	router, _ := newTestRouter(t)
	createBook(t, router, "Dune", "1", 1)
	createBook(t, router, "Dune Messiah", "2", 1)
	createBook(t, router, "Neuromancer", "3", 1)

	rec := do(t, router, http.MethodGet, "/books?q=dune&page=1&page_size=1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var page struct {
		Items []Book `json:"items"`
		Total int    `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestValidationDuplicateIsbnAndMissingBook(t *testing.T) {
	router, _ := newTestRouter(t)
	createBook(t, router, "Dune", "1", 1)

	if rec := do(t, router, http.MethodPost, "/books", `{"title":"","author":"A","isbn":"2","total_copies":1}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing title: %d", rec.Code)
	}
	if rec := do(t, router, http.MethodPost, "/books", `{"title":"X","author":"A","isbn":"1","total_copies":1}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate isbn: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, router, http.MethodGet, "/books/99", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", rec.Code)
	}
	if rec := do(t, router, http.MethodGet, "/books/abc", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id: %d", rec.Code)
	}
}

# go-catalog-service

Owns the books and how many copies are on the shelf. SQLite with `sqlx`, migrations
applied at start-up with `golang-migrate`.

```sh
DB_PATH=catalog.db PORT=8082 go run .
```

| Route | Behaviour |
|---|---|
| `GET /health` | liveness |
| `POST /books` | create; `409` on a duplicate ISBN |
| `GET /books?q=&page=&page_size=` | `{items,total,page,page_size}`, search on title and author |
| `GET/PUT/DELETE /books/{id}` | `404` when absent, `400` for a non-numeric id |
| `POST /books/{id}/reserve` | one copy off the shelf; `409` when none are free |
| `POST /books/{id}/release` | one copy back; `409` when every copy is already back |

`PUT` refuses a `total_copies` that would leave fewer copies on the shelf than are
currently borrowed.

Run the tests with `go test ./...`.

# go-catalog-service

Owns the books and how many copies of each are on the shelf.

```sh
PORT=8082 go run .
```

| Route | Behaviour |
|---|---|
| `GET /health` | `{"status":"ok"}` when the process is up |
| `GET /books/{id}` | The book as `{"id","title","author","status"}`, where `status` is `available` or `on_loan`; 404 `{"error":"book not found"}` when the id is unknown |
| `POST /books/{id}/borrow` | Marks the book `on_loan` and returns it; 404 `{"error":"book not found"}` when the id is unknown, 409 `{"error":"book already on loan"}` when it is already out |
| `POST /books/{id}/return` | Marks the book `available` and returns it; 404 `{"error":"book not found"}` when the id is unknown, 409 `{"error":"book not on loan"}` when it is already on the shelf |

The shelf is seeded in code with three available books. Borrowing sets a book
`on_loan` and returning it puts it back to `available`.

## Request correlation

The API gateway mints an `X-Request-ID` for every inbound request and forwards it
unchanged to this service. When the header is present the catalog service echoes it
back on the response and includes it in its log line:

```
request_id=<id> method=<method> path=<path> status=<status>
```

The service does not generate an ID itself — a request that arrives without the
header is served normally and logged with an empty `request_id=`.

Run the tests with `go test ./...`.

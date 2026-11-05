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

The shelf is seeded in code with three available books. Returning a book is not
implemented yet, so a borrowed book stays `on_loan` for the life of the process.

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

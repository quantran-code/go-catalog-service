# go-catalog-service

Owns the books and how many copies of each are on the shelf.

```sh
PORT=8082 go run .
```

| Route | Behaviour |
|---|---|
| `GET /health` | `{"status":"ok"}` when the process is up |

Run the tests with `go test ./...`.

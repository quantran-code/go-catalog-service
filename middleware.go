package main

import (
	"log"

	"github.com/labstack/echo/v4"
)

// RequestIDHeader is the wire header carrying the correlation ID minted by the gateway.
const RequestIDHeader string = "X-Request-ID"

// requestIDContextKey is the echo context key the request ID is stored under.
const requestIDContextKey string = "request_id"

// RequestIDMiddleware echoes the incoming request ID and logs one line per request.
// The gateway is the generator, so no ID is minted when the header is absent.
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(RequestIDHeader)
			c.Set(requestIDContextKey, id)
			if id != "" {
				c.Response().Header().Set(RequestIDHeader, id)
			}

			err := next(c)

			log.Printf("request_id=%s method=%s path=%s status=%d",
				id, c.Request().Method, c.Request().URL.Path, c.Response().Status)
			return err
		}
	}
}

// RequestID returns the request ID stored by RequestIDMiddleware, or "" when the
// middleware did not run.
func RequestID(c echo.Context) string {
	id, ok := c.Get(requestIDContextKey).(string)
	if !ok {
		return ""
	}
	return id
}

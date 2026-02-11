// Package api This file is generated ONCE as a starting point and will NOT be overwritten.
// Modify it freely to add your middleware logic.
// To regenerate, delete this file or set generate.handler.output.overwrite: true in config.
//
// Fiber provides many built-in middleware: recover, logger, cors, requestid, etc.
// See: https://docs.gofiber.io/category/-middleware
//
// This file shows how to write custom middleware using fiber.Handler.
package api

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

// ExampleMiddleware demonstrates a custom fiber.Handler middleware.
// It logs before and after each request.
func ExampleMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		log.Printf("before: %s %s", c.Method(), c.Path())
		err := c.Next()
		log.Printf("after: %s %s status=%d", c.Method(), c.Path(), c.Response().StatusCode())
		return err
	}
}

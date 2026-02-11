// Package api This file is generated ONCE as a starting point and will NOT be overwritten.
// Modify it freely to add your middleware logic.
// To regenerate, delete this file or set generate.handler.output.overwrite: true in config.
//
// Gin provides built-in middleware via gin.Default() which includes Logger and Recovery.
// Additional middleware: gin.BasicAuth(), gin.ErrorLogger(), etc.
// See: https://gin-gonic.com/docs/examples/custom-middleware/
//
// This file shows how to write custom middleware using gin.HandlerFunc.
package api

import (
	"log"

	"github.com/gin-gonic/gin"
)

// ExampleMiddleware demonstrates a custom gin.HandlerFunc middleware.
// It logs before and after each request.
func ExampleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("before: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Printf("after: %s %s status=%d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

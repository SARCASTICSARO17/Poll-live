package utils

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Success writes a consistent success response: {"success": true, "data": ...}
func Success(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"success": true, "data": data})
}

// Error writes a consistent error response: {"success": false, "message": ...}
// Internal errors are logged and never exposed to the client verbatim.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": false, "message": message})
}

// InternalError logs the underlying cause and returns a generic 500 message.
func InternalError(c *gin.Context, err error) {
	log.Printf("internal error: %v", err)
	Error(c, http.StatusInternalServerError, "internal server error")
}
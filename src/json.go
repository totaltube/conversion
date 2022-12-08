package main

import (
	"github.com/gin-gonic/gin"
)

func errorJSON(c *gin.Context, message string) {
	c.JSON(200, M{"status": "error", "message": message, "version": version})
}
func successJSON(c *gin.Context, value interface{}) {
	c.JSON(200, M{"status": "OK", "value": value})
}

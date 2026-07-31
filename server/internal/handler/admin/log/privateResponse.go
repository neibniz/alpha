package log

import "github.com/gin-gonic/gin"

func privateLogResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store, max-age=0")
	c.Header("Pragma", "no-cache")
}

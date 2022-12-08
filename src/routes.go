package main

import (
	"github.com/gin-gonic/gin"
)

func setRoutes(app *gin.Engine) {
	app.GET("/", func(c *gin.Context) {
		c.JSON(200, M{"success": true})
	})
	app.GET("/status", statusHandler)
	app.GET("/download", downloadHandler)
	app.GET("/delete", deleteHandler)
	app.GET("/delete-all", deleteAllHandler)
	app.GET("/copy", copyHandler)
	app.POST("/convert", convertHandler)
	app.POST("/make-thumbs", makeThumbsHandler)
	app.POST("/make-images", makeImagesHandler)
	app.POST("/make-video", makeVideoHandler)
	app.POST("/video-info", videoInfoHandler)
}

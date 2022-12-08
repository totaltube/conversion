package main

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/totaltube/conversion/helpers"
)

func copyHandler(c *gin.Context) {
	contentId, err := strconv.ParseUint(c.Request.URL.Query().Get("content_id"), 10, 64)
	if err != nil || contentId == 0 {
		errorJSON(c, "no content_id provided")
		return
	}
	from := c.Request.URL.Query().Get("from")
	to := c.Request.URL.Query().Get("to")
	if from == "" || to == "" {
		errorJSON(c, "no files specified for copying")
		return
	}
	copyFrom := filepath.Join(conversionPath, strconv.FormatUint(contentId, 10), from)
	copyTo := filepath.Join(conversionPath, strconv.FormatUint(contentId, 10), to)
	if !helpers.FileExists(copyFrom) {
		errorJSON(c, "file does not exists")
		return
	}
	data, err := ioutil.ReadFile(copyFrom)
	if err != nil {
		log.Println(err)
		errorJSON(c, "can't read file: "+err.Error())
		return
	}
	err = ioutil.WriteFile(copyTo, data, 0755)
	if err != nil {
		log.Println(err)
		errorJSON(c, "can't copy to "+copyTo+": "+err.Error())
		return
	}
	successJSON(c, "")
}

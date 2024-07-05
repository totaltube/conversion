package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func deleteHandler(c *gin.Context) {
	contentId, err := strconv.ParseUint(c.Request.URL.Query().Get("content_id"), 10, 64)
	if err != nil || contentId == 0 {
		errorJSON(c, "no content_id provided")
		return
	}
	file := c.Request.URL.Query().Get("file")
	if file == "" {
		// Delete all dir with content
		toDeleteDir := filepath.Join(conversionPath, strconv.FormatUint(contentId, 10))
		err = os.RemoveAll(toDeleteDir)
		if err != nil {
			log.Println("error deleting directory", toDeleteDir)
		}
		successJSON(c, "")
		return
	}
	if strings.Contains(file, "..") {
		errorJSON(c, "disabled characters in file name")
		return
	}
	workingPath := filepath.Join(conversionPath, strconv.FormatUint(contentId, 10))
	fullFileName := filepath.Join(workingPath, file)
	err = os.RemoveAll(fullFileName)
	if err != nil {
		log.Println("error deleting file", fullFileName, ":", err)
	}
	successJSON(c, "")
}


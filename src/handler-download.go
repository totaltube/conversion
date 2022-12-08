package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/totaltube/conversion/helpers"
)

func downloadHandler(c *gin.Context) {
	contentId, err := strconv.ParseUint(c.Request.URL.Query().Get("content_id"), 10, 64)
	if err != nil || contentId == 0 {
		errorJSON(c, "no content_id provided")
		return
	}
	file := c.Request.URL.Query().Get("file")
	if file == "" {
		errorJSON(c, "no file provided")
		return
	}
	if strings.Contains(file, "..") {
		errorJSON(c, "disabled characters in file name")
		return
	}
	workingPath := filepath.Join(conversionPath, strconv.FormatUint(contentId, 10))
	fullFileName := filepath.Join(workingPath, file)
	if !helpers.FileExists(fullFileName) {
		errorJSON(c, "file "+fullFileName+" does not exists")
		return
	}
	openFile, err := os.Open(fullFileName)
	if err != nil {
		errorJSON(c, "can't open file "+fullFileName)
		return
	}
	defer openFile.Close()
	fileHeader := make([]byte, 512)
	// Copy the headers into the fileHeader buffer
	_, _ = openFile.Read(fileHeader)
	// Get content type of file
	fileContentType := http.DetectContentType(fileHeader)

	// Get the file size
	fileStat, _ := openFile.Stat()                     // Get info from file
	fileSize := strconv.FormatInt(fileStat.Size(), 10) // Get file size as a string

	// Send the headers
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(fullFileName))
	c.Header("Content-Type", fileContentType)
	c.Header("Content-Length", fileSize)

	// Send the file
	// We read 512 bytes from the file already, so we reset the offset back to 0
	_, _ = openFile.Seek(0, 0)
	_, _ = io.Copy(c.Writer, openFile) // 'Copy' the file to the client
}

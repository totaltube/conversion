package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var dirRegex = regexp.MustCompile(`\d+`)

func deleteAllHandler(c *gin.Context) {
	expire, err := strconv.ParseUint(c.Request.URL.Query().Get("expire"), 10, 64)
	if err != nil || expire == 0 {
		expire = 86400
	}
	expireTime := time.Now().Add(-time.Second * time.Duration(expire))
	matches, err := filepath.Glob(filepath.Join(conversionPath, "*"))
	if err != nil {
		errorJSON(c, err.Error())
		return
	}
	for _, d := range matches {
		if !dirRegex.MatchString(d) {
			// deleting only dirs with digits - global ID
			continue
		}
		if info, err := os.Stat(d); err == nil {
			if info.IsDir() && info.ModTime().Before(expireTime) {
				_ = os.RemoveAll(d)
			}
		}
	}
	successJSON(c, "")
}


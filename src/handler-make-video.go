package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/totaltube/conversion/queries"
	"github.com/totaltube/conversion/types"
)

func makeVideoHandler(c *gin.Context) {
	var params types.MakeVideoRequest
	err := c.BindJSON(&params)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}

	var tmpDir string
	tmpDir, err = os.MkdirTemp(conversionPath, "make_video_")
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	defer os.RemoveAll(tmpDir)
	var sourceServer *types.S3Server
	if sourceServer, err = types.S3FromURL(params.Source); err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": "wrong source server url: " + err.Error()})
		return
	}
	/*var hostPort = strings.Split(sourceServer.Endpoint, ":")
	if hostPort[0] == "localhost" || hostPort[0] == "127.0.0.1" {
		hostPort[0] = "host.docker.internal"
		sourceServer.Endpoint = strings.Join(hostPort, ":")
	}*/
	var destinationServer *types.S3Server
	if destinationServer, err = types.S3FromURL(params.Destination); err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": "wrong destination server url: " + err.Error()})
		return
	}
	/*hostPort = strings.Split(destinationServer.Endpoint, ":")
	if hostPort[0] == "localhost" || hostPort[0] == "127.0.0.1" {
		hostPort[0] = "host.docker.internal"
		destinationServer.Endpoint = strings.Join(hostPort, ":")
	}*/
	var sourceFileNames []string
	if sourceFileNames, err = queries.StorageList(c, sourceServer, sourceServer.ObjectName); err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	_ = os.MkdirAll(filepath.Join(tmpDir, "sources"), os.ModePerm)
	for _, s := range sourceFileNames {
		err = queries.StorageFileGet(c, sourceServer, s, filepath.Join(tmpDir, "sources", filepath.Base(s)))
		if err != nil {
			log.Println(err)
			c.JSON(200, M{"success": false, "value": err.Error()})
			return
		}
	}
	filenames, _ := filepath.Glob(filepath.Join(tmpDir, "sources", "*"))
	var sourceNames = make([]string, 0, len(filenames))
	for _, filename := range filenames {
		var mimeType string
		ext := filepath.Ext(filename)
		if ext == ".mp4" {
			mimeType = "video/mp4"
		} else if ext == ".webm" {
			mimeType = "video/webm"
		} else {
			m, err := mimetype.DetectFile(filename)
			if err != nil {
				log.Println(err)
			}
			mimeType = m.String()
		}
		if lo.Contains(videoTypes, mimeType) {
			sourceNames = append(sourceNames, filename)
		} else {
			log.Println("wrong mime type - ", mimeType, " for file ", filename)
		}
	}

	if len(sourceNames) == 0 {
		log.Println("Filenames: ", filenames, "Source file names", sourceFileNames)
		c.JSON(200, M{"success": false, "value": "no video source files found"})
		return
	}
	_ = os.MkdirAll(filepath.Join(tmpDir, "result"), os.ModePerm)
	var info types.ContentVideoInfo
	info, err = ConvertVideo(sourceNames, tmpDir, filepath.Join(tmpDir, "result"), params.Format)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	// Done. Uploading to the server
	var success = false
	defer func() {
		if !success {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
			defer cancel()
			list, err1 := queries.StorageList(ctx, destinationServer, destinationServer.ObjectName)
			if err1 != nil {
				log.Println(err1)
				return
			}
			for _, entry := range list {
				if strings.HasPrefix(path.Base(entry), fmt.Sprintf("video-%s.", params.Format.Name)) ||
					strings.HasPrefix(path.Base(entry), fmt.Sprintf("poster-%s.", params.Format.Name)) ||
					strings.HasPrefix(path.Base(entry), fmt.Sprintf("timeline-%s.", params.Format.Name)) {
					err1 = queries.StorageDelete(ctx, destinationServer, entry)
					if err1 != nil {
						log.Println(err1)
					}
				}
			}
		}
	}()
	var resultFiles []string
	resultFiles, _ = filepath.Glob(filepath.Join(tmpDir, "result", "*"))
	for _, f := range resultFiles {
		objectName := path.Join(destinationServer.ObjectName, filepath.Base(f))
		err = queries.StorageFileUpload(c, destinationServer, f, objectName)
		if err != nil {
			log.Println(err)
			c.JSON(200, M{"success": false, "value": err.Error()})
			return
		}
	}
	success = true
	c.JSON(200, M{"success": true, "value": info})
}

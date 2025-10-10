package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/totaltube/conversion/queries"
	"github.com/totaltube/conversion/types"
)

func createPreviewHandler(c *gin.Context) {
	var params types.CreatePreviewRequest
	err := c.BindJSON(&params)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}

	var tmpDir string
	tmpDir, err = os.MkdirTemp(conversionPath, "create_preview_")
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

	var destinationServer *types.S3Server
	if destinationServer, err = types.S3FromURL(params.Destination); err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": "wrong destination server url: " + err.Error()})
		return
	}

	var sourceFileInfos []queries.FileInfo
	if sourceFileInfos, err = queries.StorageListWithSort(c, sourceServer, sourceServer.ObjectName, "size"); err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}

	// Find the largest mp4, webm, avif file that starts with "video-"
	var selectedFileInfo *queries.FileInfo
	for _, fileInfo := range sourceFileInfos {
		baseName := filepath.Base(fileInfo.Name)
		ext := strings.ToLower(filepath.Ext(baseName))
		if (ext == ".mp4" || ext == ".webm" || ext == ".avif") && strings.HasPrefix(baseName, "video-") {
			selectedFileInfo = &fileInfo
			break // Since list is sorted by size descending, first match is the largest
		}
	}

	if selectedFileInfo == nil {
		c.JSON(200, M{"success": false, "value": "no suitable video files found (mp4/webm/avif starting with video-)"})
		return
	}

	// Download the selected file
	_ = os.MkdirAll(filepath.Join(tmpDir, "sources"), os.ModePerm)
	localPath := filepath.Join(tmpDir, "sources", filepath.Base(selectedFileInfo.Name))
	err = queries.StorageFileGet(c, sourceServer, selectedFileInfo.Name, localPath)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": "failed to download video file: " + err.Error()})
		return
	}

	// Use the selected file directly since we've already filtered it
	videoSourceFile := localPath

	// Create video preview
	previewFileName := fmt.Sprintf("video-preview-%s.mp4", params.Format.Name)
	previewFilePath := filepath.Join(tmpDir, previewFileName)
	err = CreateVideoPreview(videoSourceFile, tmpDir, params.Format, previewFilePath)
	if err != nil {
		log.Printf("Failed to create video preview: %v", err)
		c.JSON(200, M{"success": false, "value": "failed to create video preview: " + err.Error()})
		return
	}

	// Upload video preview to destination
	objectName := path.Join(destinationServer.ObjectName, previewFileName)
	err = queries.StorageFileUpload(c, destinationServer, previewFilePath, objectName)
	if err != nil {
		log.Printf("Failed to upload video preview: %v", err)
		c.JSON(200, M{"success": false, "value": "failed to upload video preview: " + err.Error()})
		return
	}

	c.JSON(200, M{"success": true})
}

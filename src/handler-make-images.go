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
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/totaltube/conversion/helpers"
	"github.com/totaltube/conversion/queries"
	"github.com/totaltube/conversion/types"
)

var errorLowRes = errors.New("too low res source image")

func makeImagesHandler(c *gin.Context) {
	var params types.MakeImagesRequest
	err := c.BindJSON(&params)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	if params.Format.Command == "" {
		params.Format.Command = "%MAGICK_PATH%convert %SOURCE_FILE% -thumbnail %SIZE%^ -gravity center -extent %SIZE% -quality 92 %RESULT_FILE%"
	}
	var tmpDir string
	tmpDir, err = os.MkdirTemp(conversionPath, "make_images_")
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	tmpDir, _ = filepath.Abs(tmpDir)
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
	var numCreated int64
	var processImage = func(imageFile string) (size string, previewSize string, err error) {
		var width int
		var height int
		width, height, err = helpers.GetImageDimensions(imageFile)
		if err != nil {
			err = errors.Wrap(err, "wrong image format")
			log.Println(err)
			return
		}
		size = fmt.Sprintf("%dx%d", params.Format.Size.Width, params.Format.Size.Height)
		previewSize = fmt.Sprintf("%dx%d", params.Format.PreviewSize.Width, params.Format.PreviewSize.Height)
		if (params.Format.Size.Width < params.Format.Size.Height && width > height) || (params.Format.Size.Width > params.Format.Size.Height && width < height) {
			size = fmt.Sprintf("%dx%d", params.Format.Size.Height, params.Format.Size.Width)
			previewSize = fmt.Sprintf("%dx%d", params.Format.PreviewSize.Height, params.Format.PreviewSize.Width)
		}
		var minWidth = params.Format.MinSourceSize.Width
		var minHeight = params.Format.MinSourceSize.Height
		if minWidth > minHeight && width < height {
			minWidth, minHeight = minHeight, minWidth
		} else if minWidth < minHeight && width > height {
			minWidth, minHeight = minHeight, minWidth
		}
		if width < int(minWidth) || height < int(minHeight) {
			err = errorLowRes
			return
		}
		newPath := filepath.Join(tmpDir,
			fmt.Sprintf("image-%s.%d.%s", params.Format.Name, numCreated, params.Format.Type))
		err = ConvertImage(imageFile, params.Format.Command, newPath, size)
		if err != nil {
			return
		}
		width, height, err = helpers.GetImageDimensions(newPath)
		if err != nil {
			_ = os.Remove(newPath)
			return
		}
		size = fmt.Sprintf("%dx%d", width, height)
		previewNewPath := filepath.Join(tmpDir,
			fmt.Sprintf("preview-%s.%d.%s", params.Format.Name, numCreated, params.Format.Type))
		err = ConvertImage(imageFile, params.Format.Command, previewNewPath, previewSize)
		if err != nil {
			_ = os.Remove(newPath)
			return
		}
		width, height, err = helpers.GetImageDimensions(previewNewPath)
		if err != nil {
			_ = os.Remove(newPath)
			_ = os.Remove(previewNewPath)
			log.Println(err)
			return
		}
		previewSize = fmt.Sprintf("%dx%d", width, height)
		numCreated++
		return
	}
	/*if params.Format.MaxAmount > 0 {
		filenames = lo.Shuffle(filenames)
	}*/
	var items = make([]string, 0, 300)
	var previewItems = make([]string, 0, 300)
	for _, f := range filenames {
		if params.Format.MaxAmount > 0 && numCreated >= params.Format.MaxAmount {
			break
		}
		m, _ := mimetype.DetectFile(f)
		mimeType := m.String()
		if lo.Contains(imageTypes, mimeType) {
			var size string
			var previewSize string
			size, previewSize, err = processImage(f)
			if err != nil {
				if err == errorLowRes {
					continue
				}
				log.Println(err)
				c.JSON(200, M{"success": false, "value": err.Error()})
				return
			}
			items = append(items, size)
			previewItems = append(previewItems, previewSize)
			continue
		}
		if lo.Contains(videoTypes, mimeType) {
			// The source is video, making needed frames
			_ = os.MkdirAll(filepath.Join(tmpDir, "frames"), os.ModePerm)
			maxFrames := params.Format.MaxAmount - numCreated
			if maxFrames < 0 {
				maxFrames = 0
			}
			err = doExtractFrames2(f, filepath.Join(tmpDir, "frames"), maxFrames, params.Format.MinTimeInterval, true)
			if err != nil {
				log.Println(err)
				c.JSON(200, M{"success": false, "value": err.Error()})
				return
			}
			var frameFiles []string
			frameFiles, _ = filepath.Glob(filepath.Join(tmpDir, "frames", "*"))
			/*if params.Format.MaxAmount > 0 {
				frameFiles = lo.Shuffle(frameFiles)
			}*/
			for _, f := range frameFiles {
				if params.Format.MaxAmount > 0 && numCreated >= params.Format.MaxAmount {
					break
				}
				var size string
				var previewSize string
				size, previewSize, err = processImage(f)
				if err != nil {
					if err == errorLowRes {
						continue
					}
					log.Println(err)
					c.JSON(200, M{"success": false, "value": err.Error()})
					return
				}
				items = append(items, size)
				previewItems = append(previewItems, previewSize)
			}
		}
	}
	if params.Format.MinAmount > 0 && len(items) < int(params.Format.MinAmount) {
		c.JSON(200, M{
			"success": false,
			"value": fmt.Sprintf("Number of created images (%d) is below minimum amount %d",
				len(items), params.Format.MinAmount)})
		return
	}
	// Done. Uploading to the server.
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
				if strings.HasPrefix(path.Base(entry), fmt.Sprintf("image-%s.", params.Format.Name)) ||
					strings.HasPrefix(path.Base(entry), fmt.Sprintf("preview-%s.", params.Format.Name)) {
					err1 = queries.StorageDelete(ctx, destinationServer, entry)
					if err1 != nil {
						log.Println(err1)
					}
				}
			}
		}
	}()
	var resultFiles []string
	resultFiles, _ = filepath.Glob(filepath.Join(tmpDir, "image-*"))
	for _, f := range resultFiles {
		objectName := path.Join(destinationServer.ObjectName, filepath.Base(f))
		err = queries.StorageFileUpload(c, destinationServer, f, objectName)
		if err != nil {
			log.Println(err)
			c.JSON(200, M{"success": false, "value": err.Error()})
			return
		}
	}
	resultFiles, _ = filepath.Glob(filepath.Join(tmpDir, "preview-*"))
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
	c.JSON(200, M{"success": true, "value": M{"items": items, "preview_items": previewItems}})
}

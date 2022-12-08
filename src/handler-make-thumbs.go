package main

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
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
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/totaltube/conversion/queries"
	"github.com/totaltube/conversion/types"
)

var videoTypes = []string{
	"video/ogv", "video/mpeg", "video/mp4", "video/quicktime", "video/webm", "video/x-flv",
	"video/x-matroska", "video/3gpp", "video/3gpp2",
}
var imageTypes = []string{
	"image/jpeg", "image/png", "image/webp", "image/gif",
}

func makeThumbsHandler(c *gin.Context) {
	var params types.MakeThumbsRequest
	err := c.BindJSON(&params)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	var tmpDir string
	tmpDir, err = ioutil.TempDir(conversionPath, "make_thumbs_")
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
	var hostPort = strings.Split(sourceServer.Endpoint, ":")
	if hostPort[0] == "localhost" || hostPort[0] == "127.0.0.1" {
		hostPort[0] = "host.docker.internal"
		sourceServer.Endpoint = strings.Join(hostPort, ":")
	}
	var destinationServer *types.S3Server
	if destinationServer, err = types.S3FromURL(params.Destination); err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": "wrong destination server url: " + err.Error()})
		return
	}
	hostPort = strings.Split(destinationServer.Endpoint, ":")
	if hostPort[0] == "localhost" || hostPort[0] == "127.0.0.1" {
		hostPort[0] = "host.docker.internal"
		destinationServer.Endpoint = strings.Join(hostPort, ":")
	}
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
	size := fmt.Sprintf("%dx%d", params.Format.Size.Width, params.Format.Size.Height)
	retina := false
	var processImage = func(imageFile string) error {
		reader, _ := os.Open(imageFile)
		defer reader.Close()
		im, _, err := image.DecodeConfig(reader)
		if err != nil {
			return errors.Wrap(err, "wrong image format")
		}
		if im.Width < int(params.Format.MinSourceSize.Width) || im.Height < int(params.Format.MinSourceSize.Height) {
			return nil
		}
		newPath := filepath.Join(tmpDir,
			fmt.Sprintf("thumb-%s.%d.%s", params.Format.Name, numCreated, params.Format.Type))
		err = ConvertImage(imageFile, params.Format.Command, newPath, size)
		if err != nil {
			return err
		}
		var info os.FileInfo
		info, err = os.Stat(newPath)
		if err != nil {
			return err
		}
		imageSize := info.Size()
		if imageSize < params.Format.MinSize {
			return nil
		}
		if numCreated == 0 && params.Format.Retina {
			// Checking if we can make retina (highres) image
			if im.Width >= int(params.Format.RetinaMinSourceSize.Width) && im.Height >= int(params.Format.RetinaMinSourceSize.Height) {
				retina = true
			}
		}
		if retina {
			retinaSize := fmt.Sprintf("%dx%d", params.Format.Size.Width*2, params.Format.Size.Height*2)
			err = ConvertImage(imageFile, params.Format.Command, filepath.Join(tmpDir,
				fmt.Sprintf("thumb-%s.%d@2x.%s", params.Format.Name, numCreated, params.Format.Type),
			), retinaSize)
			if err != nil {
				return err
			}
		}
		numCreated++
		return nil
	}
	if params.MaxThumbs > 0 {
		filenames = lo.Shuffle(filenames)
	}
	for _, f := range filenames {
		if params.Format.MaxThumbs > 0 && numCreated >= params.Format.MaxThumbs {
			break
		}
		if params.MaxThumbs > 0 && numCreated >= params.MaxThumbs {
			break
		}
		m, _ := mimetype.DetectFile(f)
		mimeType := m.String()
		if lo.Contains(imageTypes, mimeType) {
			err = processImage(f)
			if err != nil {
				log.Println(err)
				c.JSON(200, M{"success": false, "value": err.Error()})
				return
			}
			continue
		}
		if lo.Contains(videoTypes, mimeType) {
			// The source is vide, making frames
			_ = os.MkdirAll(filepath.Join(tmpDir, "frames"), os.ModePerm)
			maxFrames := params.Format.MaxThumbs - numCreated
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
			if params.MaxThumbs > 0 {
				frameFiles = lo.Shuffle(frameFiles)
			}
			for _, f := range frameFiles {
				if params.Format.MaxThumbs > 0 && numCreated >= params.Format.MaxThumbs {
					break
				}
				if params.MaxThumbs > 0 && numCreated >= params.MaxThumbs {
					break
				}
				err = processImage(f)
				if err != nil {
					log.Println(err)
					c.JSON(200, M{"success": false, "value": err.Error()})
					return
				}
			}
		}
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
				if strings.HasPrefix(path.Base(entry), fmt.Sprintf("thumb-%s.", params.Format.Name)) {
					err1 = queries.StorageDelete(ctx, destinationServer, entry)
					if err1 != nil {
						log.Println(err1)
					}
				}
			}
		}
	}()
	var resultFiles []string
	resultFiles, _ = filepath.Glob(filepath.Join(tmpDir, "thumb-*"))
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
	c.JSON(200, M{"success": true, "value": M{"num_created": numCreated, "retina": retina}})
}

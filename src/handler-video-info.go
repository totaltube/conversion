package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/totaltube/conversion/queries"
	"github.com/totaltube/conversion/types"
)

func videoInfoHandler(c *gin.Context) {
	var params types.VideoInfoRequest
	err := c.BindJSON(&params)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}

	var tmpDir string
	tmpDir, err = ioutil.TempDir(conversionPath, "video_info_")
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
	var sourceFilename = filepath.Join(tmpDir, filepath.Base(sourceServer.ObjectName))
	err = queries.StorageFileGet(c, sourceServer, sourceServer.ObjectName, sourceFilename)
	if err != nil {
		log.Println(err)
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	m, _ := mimetype.DetectFile(sourceFilename)
	mimeType := m.String()
	if !lo.Contains(videoTypes, mimeType) {
		c.JSON(200, M{"success": false, "value": "source file is not a video"})
		return
	}
	cmd := exec.Command("ffprobe", sourceFilename, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", "-print_format", "json")
	var out []byte
	out, err = cmd.CombinedOutput()
	if err != nil {
		err = errors.New("can't run ffprobe: " + err.Error())
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	var fileFormat types.FileFormat
	err = json.Unmarshal(out, &fileFormat)
	if err != nil {
		err = errors.New("can't parse ffprobe output: " + err.Error())
		c.JSON(200, M{"success": false, "value": err.Error()})
		return
	}
	fileFormat.Format.Filename = ""
	c.JSON(200, M{"success": true, "value": fileFormat})
}

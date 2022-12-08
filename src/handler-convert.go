package main

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func convertHandler(c *gin.Context) {
	var err error
	var isMultipart = strings.Contains(strings.ToLower(c.Request.Header.Get("Content-Type")), "multipart/form-data")
	if isMultipart {
		err = c.Request.ParseMultipartForm(32 << 20)
		if err != nil {
			errorJSON(c, "can't parse multipart form: "+err.Error())
			return
		}
	} else {
		err = c.Request.ParseForm()
		if err != nil {
			errorJSON(c, "can't parse form: "+err.Error())
			return
		}
	}
	contentId, err := strconv.ParseUint(c.Request.FormValue("content_id"), 10, 64)
	if err != nil || contentId == 0 {
		errorJSON(c, "no content_id provided")
		return
	}
	action := c.Request.FormValue("action")
	rawFormat := c.Request.FormValue("format")
	if rawFormat == "" {
		errorJSON(c, "format not specified")
		return
	}
	workingPath := filepath.Join(conversionPath, strconv.FormatUint(contentId, 10))
	tempPath := filepath.Join(workingPath, "temp")
	sourcesPath := filepath.Join(workingPath, "sources")
	err = os.MkdirAll(tempPath, 0755)
	if err != nil {
		errorJSON(c, "can't create temp directory: "+err.Error())
		return
	}
	defer os.RemoveAll(tempPath)
	err = os.MkdirAll(sourcesPath, 0755)
	if err != nil {
		errorJSON(c, "can't create sources directory: "+err.Error())
		return
	}
	var files []string
	var i int64
	if isMultipart {
		for _, fHeaders := range c.Request.MultipartForm.File {
			for _, f := range fHeaders {
				fName := filepath.Base(f.Filename)
				i++
				var fp multipart.File
				if fp, err = f.Open(); err != nil {
					errorJSON(c, "can't open uploaded file: "+err.Error())
					return
				}
				var outfile *os.File
				if outfile, err = os.Create(filepath.Join(sourcesPath, fName)); err != nil {
					_ = fp.Close()
					errorJSON(c, "can't create file in "+sourcesPath+": "+err.Error())
					return
				}
				_, err = io.Copy(outfile, fp)
				if err != nil {
					errorJSON(c, "can't copy uploaded file into sources directory: "+err.Error())
					return
				}
				err = fp.Close()
				if err != nil {
					errorJSON(c, "can't close file: "+err.Error())
					return
				}
				err = outfile.Close()
				if err != nil {
					errorJSON(c, "can't close out file: "+err.Error())
					return
				}
				files = append(files, fName)
			}
		}
	}
	var formFiles = make([]string, 0, 1)
	for k, v := range c.Request.Form {
		if strings.HasPrefix(k, "files[") {
			formFiles = append(formFiles, v[0])
		}
	}
	if len(files) == 0 {
		files = formFiles
	}
	if len(files) == 0 {
		errorJSON(c, "no files provided for conversion")
		return
	}
	switch action {
	case "convert-image":
		var format ImageFormat
		err = json.Unmarshal([]byte(rawFormat), &format)
		if err != nil {
			errorJSON(c, "can't decode format from json: "+err.Error())
			return
		}
		var resultFiles []string
		resultFiles, err = doConvertImage(files, workingPath, format)
		if err != nil {
			errorJSON(c, err.Error())
			return
		}
		c.JSON(200, M{"status": "OK", "files": resultFiles, "log": ""})
		return
	case "convert-video":
		var format VideoFormat
		err = json.Unmarshal([]byte(rawFormat), &format)
		if err != nil {
			errorJSON(c, "can't decode format from json: "+err.Error())
			return
		}
		var resultFiles []string
		resultFiles, err = doConvertVideo(files, workingPath, format)
		if err != nil {
			errorJSON(c, err.Error())
			return
		}
		c.JSON(200, M{"status": "OK", "files": resultFiles, "log": ""})
		return
	case "extract-frames":
		var format ExtractFramesFormat
		err = json.Unmarshal([]byte(rawFormat), &format)
		if err != nil {
			errorJSON(c, "can't decode format from json: "+err.Error())
			return
		}
		var resultFiles []string
		resultFiles, err = doExtractFrames(files, workingPath, &format)
		if err != nil {
			errorJSON(c, err.Error())
			return
		}
		c.JSON(200, M{
			"status":   "OK",
			"files":    resultFiles,
			"amount":   format.Amount,
			"start":    format.Start,
			"interval": format.Interval,
			"log":      "",
		})
		return
	case "append-images":
		var format AppendFormat
		err = json.Unmarshal([]byte(rawFormat), &format)
		if err != nil {
			errorJSON(c, "can't decode format from json: "+err.Error())
			return
		}
		ext := "png"
		if format.Type != "" {
			ext = format.Type
		}
		resultFileName := "combined." + ext
		resultFile := filepath.Join(workingPath, resultFileName)
		err = AppendImages(files, resultFile, format)
		if err != nil {
			errorJSON(c, err.Error())
			return
		}
		c.JSON(200, M{"status": "OK", "files": []string{resultFileName}, "log": ""})
		return
	case "create-timeline":
		var format TimelineFormat
		err = json.Unmarshal([]byte(rawFormat), &format)
		if err != nil {
			errorJSON(c, "can't decode format from json: "+err.Error())
			return
		}
		var resultFiles []string
		resultFiles, err = CreateTimeline(files, workingPath, format)
		if err != nil {
			errorJSON(c, err.Error())
			return
		}
		c.JSON(200, M{"status": "OK", "files": resultFiles, "log": ""})
		return
	default:
		errorJSON(c, "unknown action `"+action+"`")
		return
	}
}

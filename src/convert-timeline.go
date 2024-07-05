package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/totaltube/conversion/helpers"
)

func CreateTimeline(files []string, workingPath string, format TimelineFormat) (resultFiles []string, err error) {
	var fullFiles = make([]string, len(files))
	for k, file := range files {
		if !strings.HasPrefix("/", file) {
			fullFiles[k] = filepath.Join(workingPath, "sources", file)
		} else {
			fullFiles[k] = file
		}
	}
	guid := xid.New()
	processingFiles := fullFiles
	if format.Merge {
		// Combining all screenshots into one file
		var resizedFiles = make([]string, len(fullFiles))
		for k, file := range fullFiles {
			resizedFile := filepath.Join(workingPath, filepath.Base(file)+"-"+format.Size+".png")
			cmd := exec.Command("convert", file,
				"-thumbnail", format.Size+"^", "-gravity", "center", "-extent", format.Size, resizedFile)
			var out []byte
			out, err = cmd.CombinedOutput()
			if err != nil {
				log.Println(string(out))
				return
			}
			if !helpers.FileExists(resizedFile) {
				err = errors.New("can't create resized file from frame - result file not exists")
				return
			}
			resizedFiles[k] = resizedFile
		}
		resultFile := filepath.Join(workingPath, "timeline-combined-"+guid.String()+".png")
		args := []string{"+append"}
		args = append(args, fullFiles...)
		args = append(args, resultFile)
		cmd := exec.Command("convert", args...)
		var out []byte
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Println(string(out))
			err = errors.New("can't combine resized timeline images: " + err.Error())
			return
		}
		if !helpers.FileExists(resultFile) {
			log.Println(string(out))
			err = errors.New("result filename of combined timeline images not exists")
			return
		}
		processingFiles = []string{resultFile}
	}
	for _, file := range processingFiles {
		ext := format.Type
		if ext == "" {
			ext = "jpg"
		}
		var origWidth, origHeight uint64
		var width, height uint64
		var size string
		var origSize string
		var resultFile string
		var reader *os.File
		if reader, err = os.Open(file); err == nil {
			var im image.Config
			im, _, err = image.DecodeConfig(reader)
			if err != nil {
				reader.Close()
				err = errors.New("can't determine size of " + file + ": " + err.Error())
				return
			}
			origWidth = uint64(im.Width)
			origHeight = uint64(im.Height)
			reader.Close()
		}
		origSize = fmt.Sprintf("%dx%d", origWidth, origHeight)
		if format.Merge {
			width = origWidth
			height = origHeight
			size = origSize
			resultFile = filepath.Join(workingPath, "timeline-"+guid.String()+"."+ext)
		} else {
			matches := sizeRegexp.FindStringSubmatch(format.Size)
			if matches == nil {
				err = errors.New("incorrect size in format - " + format.Size)
				return
			}
			width, _ = strconv.ParseUint(matches[1], 10, 64)
			height, _ = strconv.ParseUint(matches[2], 10, 64)
			size = format.Size
			resultFile = filepath.Join(workingPath, "timeline-"+guid.String()+"-"+filepath.Base(file)+"."+ext)
		}
		cmd := format.Command
		cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE%", file)
		cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE_BASE%", filepath.Base(file))
		cmd = strings.ReplaceAll(cmd, "%ORIG_SIZE%", origSize)
		cmd = strings.ReplaceAll(cmd, "%ORIG_WIDTH%", strconv.FormatUint(origWidth, 10))
		cmd = strings.ReplaceAll(cmd, "%ORIG_HEIGHT%", strconv.FormatUint(origHeight, 10))
		cmd = strings.ReplaceAll(cmd, "%SIZE%", size)
		cmd = strings.ReplaceAll(cmd, "%WIDTH%", strconv.FormatUint(width, 10))
		cmd = strings.ReplaceAll(cmd, "%HEIGHT%", strconv.FormatUint(height, 10))
		cmd = strings.ReplaceAll(cmd, "%RESULT_FILE%", resultFile)
		cmd = strings.ReplaceAll(cmd, "%RESULT_FILE_BASE%", filepath.Base(resultFile))
		cmd = strings.ReplaceAll(cmd, "%MAGICK_PATH%", "")
		cmd = strings.ReplaceAll(cmd, "%MAGICK%", "magick")
		cmds := strings.Split(cmd, " && ")
		for _, cm := range cmds {
			cmdArgs := strings.Split(cm, " ")
			c := cmdArgs[0]
			args := cmdArgs[1:]
			var argsFiltered []string
			for _, arg := range args {
				if arg != "" {
					argsFiltered = append(argsFiltered, arg)
				}
			}
			command := exec.Command(c, argsFiltered...)
			command.Dir = workingPath
			var out []byte
			out, err = command.CombinedOutput()
			if err != nil {
				log.Println(string(out))
				err = errors.New("can't run command" + c + " " + strings.Join(argsFiltered, " ") + ": " + err.Error())
				return
			}
		}
		if !helpers.FileExists(resultFile) {
			err = errors.New("timeline result file " + resultFile + " not exists after running imagemagick. Something is really wrong")
			return
		}
		resultFiles = append(resultFiles, filepath.Base(resultFile))
	}
	return
}

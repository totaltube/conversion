package main

import (
	"encoding/csv"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/totaltube/conversion/helpers"
)

func ConvertImage(sourceFile, cmd, resultFile, size string) error {
	var origWidth, origHeight uint64
	var width, height uint64
	if reader, err := os.Open(sourceFile); err == nil {
		im, _, err := image.DecodeConfig(reader)
		if err != nil {
			reader.Close()
			return errors.New("Can't open source file " + sourceFile + ": " + err.Error())
		}
		origWidth = uint64(im.Width)
		origHeight = uint64(im.Height)
		reader.Close()
	}
	var err error
	if size == "" {
		size = fmt.Sprintf("%dx%d", origWidth, origHeight)
	}
	wh := strings.Split(size, "x")
	if len(wh) != 2 {
		return errors.New("wrong format of image size - " + size)
	}
	width, err = strconv.ParseUint(wh[0], 10, 64)
	if err != nil {
		return errors.New("wrong format of image size - " + size)
	}
	height, err = strconv.ParseUint(wh[1], 10, 64)
	if err != nil {
		return errors.New("wrong format of image size - " + size)
	}
	sourceFile, _ = filepath.Abs(sourceFile)
	resultFile, _ = filepath.Abs(resultFile)
	var tempPath = filepath.Join(filepath.Dir(resultFile), "temp")
	_ = os.MkdirAll(tempPath, 0755)
	defer os.RemoveAll(tempPath)
	cmd = strings.ReplaceAll(cmd, "%MAGICK_PATH%", "")
	cmd = strings.ReplaceAll(cmd, "%MAGICK%", "magick")
	cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE%", sourceFile)
	cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE_BASE%", filepath.Base(sourceFile))
	cmd = strings.ReplaceAll(cmd, "%RESULT_FILE%", resultFile)
	cmd = strings.ReplaceAll(cmd, "%RESULT_FILE_BASE%", filepath.Base(resultFile))
	cmd = strings.ReplaceAll(cmd, "%ORIG_SIZE%", fmt.Sprintf("%dx%d", origWidth, origHeight))
	cmd = strings.ReplaceAll(cmd, "%ORIG_WIDTH%", strconv.FormatUint(origWidth, 10))
	cmd = strings.ReplaceAll(cmd, "%ORIG_HEIGHT%", strconv.FormatUint(origHeight, 10))
	cmd = strings.ReplaceAll(cmd, "%SIZE%", size)
	cmd = strings.ReplaceAll(cmd, "%WIDTH%", strconv.FormatUint(width, 10))
	cmd = strings.ReplaceAll(cmd, "%HEIGHT%", strconv.FormatUint(height, 10))
	cmd = strings.ReplaceAll(cmd, "%TEMP_PATH%", tempPath)
	cmds := strings.Split(cmd, " && ")
	var out []byte
	for _, cm := range cmds {
		cr := csv.NewReader(strings.NewReader(cm))
		cr.Comma = ' '
		var cmdArgs []string
		cmdArgs, err = cr.Read()
		if err != nil {
			return errors.New("can't parse command " + `"` + cm + `"` + ": " + err.Error())
		}
		if len(cmdArgs) < 2 {
			return errors.New("wrong command: " + cm)
		}
		c := cmdArgs[0]
		args := cmdArgs[1:]
		var argsFiltered []string
		for _, arg := range args {
			if arg != "" {
				argsFiltered = append(argsFiltered, arg)
			}
		}
		command := exec.Command(c, argsFiltered...)
		command.Dir = filepath.Dir(resultFile)
		out, err = command.CombinedOutput()
		if err != nil {
			log.Println("Error converting image", sourceFile, ":", string(out))
			return errors.New("can't run command " + c + " " + strings.Join(argsFiltered, " ") + ": " + err.Error())
		}
	}
	syscall.Sync()
	if !helpers.FileExists(resultFile) {
		log.Println(cmd)
		log.Println(string(out))
		return errors.New("looks like result file " + resultFile + " not created from " + sourceFile)
	}
	return nil
}

func ConvertImageAction(source, destination, size string) error {
	cmd := `%MAGICK_PATH%convert %SOURCE_FILE% -thumbnail %SIZE%^ -gravity center -extent %SIZE% %RESULT_FILE%`
	if strings.HasSuffix(strings.ToLower(destination), ".jpg") ||
		strings.HasSuffix(strings.ToLower(destination), ".jpeg") {
		cmd = `%MAGICK_PATH%convert %SOURCE_FILE% -thumbnail %SIZE%^ -gravity center -extent %SIZE% -strip -interlace Plane -quality 80 %RESULT_FILE%`
	}
	if strings.Contains(source, " ") || strings.Contains(destination, " ") {
		fmt.Println("Source and destination files must not contain any spaces!")
		return errors.New("Source and destination files must not contain any spaces!")
	}
	return ConvertImage(source, cmd, destination, size)
}

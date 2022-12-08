package main

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/totaltube/conversion/helpers"
)

var sizeRegexp = regexp.MustCompile(`^(\d+)x(\d+)$`)

func doConvertImage(files []string, workingPath string, format ImageFormat) (resultFiles []string, err error) {
	if len(files) == 0 {
		err = errors.New("no files provided for converting")
		return
	}
	workingPath, err = filepath.Abs(workingPath)
	if err != nil {
		err = errors.New("wrong working path - " + workingPath + ": " + err.Error())
		return
	}
	matches := sizeRegexp.FindStringSubmatch(format.Size)
	if matches == nil {
		err = errors.New("size not set in passed format")
		return
	}

	for _, f := range files {
		fullName := filepath.Join(workingPath, "sources", f)
		if !helpers.FileExists(fullName) {
			err = errors.New("file " + fullName + " not exists")
			return
		}
		ext := format.Type
		if ext != "" && !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if ext == "" {
			ext = filepath.Ext(fullName)
		}
		resultFile := filepath.Join(workingPath, strings.TrimSuffix(filepath.Base(fullName), filepath.Ext(fullName))+ext)
		resultFiles = append(resultFiles, filepath.Base(resultFile))
		err = ConvertImage(fullName, format.Command, resultFile, format.Size)
		if err != nil {
			return
		}
	}
	return
}

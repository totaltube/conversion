package main

import (
	"path/filepath"

	"github.com/pkg/errors"
)

func doConvertVideo(files []string, workingPath string, format VideoFormat) (resultFiles []string, err error) {
	if len(files) == 0 {
		err = errors.New("no files specified")
		return
	}
	if format.Type == "" {
		format.Type = "mp4"
	}
	resultFile := filepath.Join(workingPath, "ready."+format.Type)
	resultFiles = []string{filepath.Base(resultFile)}
	var fullFilenames = make([]string, len(files))
	for k, f := range files {
		fullFilenames[k] = filepath.Join("sources", f)
	}
	err = ConvertVideoOld(fullFilenames, resultFile, format)
	return
}

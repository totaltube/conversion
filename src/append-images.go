package main

import (
	"log"
	"os/exec"
	"strconv"

	"github.com/pkg/errors"
)

func AppendImages(files []string, resultFile string, format AppendFormat) error {
	args := []string{"+append"}
	args = append(args, files...)
	if format.Quality != 0 {
		args = append(args, "-quality", strconv.FormatInt(int64(format.Quality), 10))
	}
	args = append(args, resultFile)
	cmd := exec.Command("convert", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return errors.New("can't append images: " + err.Error())
	}
	return nil
}

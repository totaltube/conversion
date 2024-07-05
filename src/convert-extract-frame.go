package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/totaltube/conversion/helpers"
)

func ExtractFrame(fileName string, seek time.Duration, outFilename string) error {
	guid := xid.New()
	tmpdir := filepath.Join(conversionPath, "tmp", guid.String())
	err := os.MkdirAll(tmpdir, 0755)
	if err != nil {
		return errors.New("can't create temporary directory " + tmpdir + ": " + err.Error())
	}
	defer os.RemoveAll(tmpdir)
	seekSeconds := strconv.FormatFloat(float64(seek.Nanoseconds())/1e+9, 'f', 2, 64)
	cmd := exec.Command("ffmpeg", "-y", "-hide_banner",
		"-ss", seekSeconds, "-i", fileName, "-vframes", "3", "-f", "image2", filepath.Join(tmpdir, "_tmp.%d.png"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return errors.New("can't extract frames from video file " + fileName + ": " + err.Error())
	}
	matches, err := filepath.Glob(filepath.Join(tmpdir, "_tmp") + "*")
	var biggestFile string
	var maxSize int
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil {
			return errors.New("can't stat frame file " + m)
		}
		if int(fi.Size()) > maxSize {
			biggestFile = m
			maxSize = int(fi.Size())
		}
	}
	if biggestFile == "" {
		return errors.New("can't find extracted frame for file " + fileName)
	}
	err = os.Rename(biggestFile, outFilename)
	if err != nil {
		return errors.Wrap(err, "can't move file "+biggestFile+" to "+outFilename)
	}
	return nil
}

func ExtractFrameAction(sourceFile, extractTime, outFilename string) error {
	dur := helpers.ParseHumanDuration(extractTime)
	if !helpers.FileExists(sourceFile) {
		return errors.New("file " + sourceFile + " not exists")
	}
	cmd := exec.Command("ffprobe", sourceFile, "-v", "quiet", "-print_format", "json", "-show_format", "-print_format", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("can't run ffprobe: " + err.Error())
	}
	var fileFormat FileFormat
	err = json.Unmarshal(out, &fileFormat)
	if err != nil {
		return errors.New("can't parse ffprobe output: " + err.Error())
	}
	duration, err := strconv.ParseFloat(fileFormat.Format.Duration, 64)
	if err != nil || duration < 3 {
		return errors.New("wrong duration of video " + sourceFile + ": " + fileFormat.Format.Duration)
	}
	if duration < float64(dur.Nanoseconds())/1e+9 {
		return errors.New("seek seconds " + strconv.FormatFloat(float64(dur.Nanoseconds())/1e+9, 'f', 2, 64) +
			" is out of video duration (" + strconv.FormatFloat(duration, 'f', 2, 64) + ")")
	}
	err = ExtractFrame(sourceFile, dur, outFilename)
	if err != nil {
		return err
	}
	return nil
}

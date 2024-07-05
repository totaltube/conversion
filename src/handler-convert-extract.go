package main

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/totaltube/conversion/helpers"
)

func doExtractFrames(files []string, workingPath string, format *ExtractFramesFormat) (resultFiles []string, err error) {
	if len(files) != 1 {
		err = errors.New("for frames extraction only one file should be specified")
		return
	}
	if !format.Single && (format.Amount == 0 || format.Interval == 0) {
		err = errors.New("amount or interval not specified")
		return
	}
	file := filepath.Join(workingPath, "sources", files[0])
	if format.Single {
		format.Start = format.TimeOffset
		format.Amount = 1
		format.Interval = 0
	} else {
		if format.Duration == 0 {
			// Getting video duration
			cmd := exec.Command("ffprobe", file, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", "-print_format", "json")
			var out []byte
			out, err = cmd.CombinedOutput()
			if err != nil {
				err = errors.New("can't run ffprobe: " + err.Error())
				return
			}
			var fileFormat FileFormat
			err = json.Unmarshal(out, &fileFormat)
			if err != nil {
				err = errors.New("can't parse ffprobe output: " + err.Error())
				return
			}
			format.Duration, _ = strconv.ParseFloat(fileFormat.Format.Duration, 64)
			if format.Duration < 3 {
				err = errors.New("video duration < 3 seconds: " + strconv.FormatFloat(format.Duration, 'f', 3, 64))
				return
			}
		}
		if format.Amount > 0 && format.Interval > 0 {
			if format.Duration/(float64(format.Amount+1)) < format.Interval {
				format.Amount = 0
			} else {
				format.Interval = 0
			}
		}
		if format.Amount == 0 {
			format.Amount = int64(math.Floor(format.Duration / format.Interval))
			if format.Amount < 1 {
				format.Amount = 1
				format.Start = format.Duration / 2
			} else {
				format.Start = (format.Duration - format.Interval*float64(format.Amount)) / 2
			}
		} else if format.Interval == 0 {
			format.Interval = format.Duration / float64(format.Amount+1)
			format.Start = format.Interval
		}
	}
	var i int64
	for i = 0; i < format.Amount; i++ {
		seek := time.Duration(format.Start+float64(i)*format.Interval) * time.Second
		resultFileName := "frame." + strconv.FormatInt(i, 10) + ".png"
		resultFiles = append(resultFiles, resultFileName)
		frameFile := filepath.Join(workingPath, "sources", resultFileName)
		err = ExtractFrame(file, seek, frameFile)
		if err != nil {
			return
		}
	}
	return
}

func doExtractFrames2(file string, destinationPath string, maxAmount int64, interval float64, randomize bool) (err error) {
	// Getting video duration
	cmd := exec.Command("ffprobe", file, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", "-print_format", "json")
	var out []byte
	out, err = cmd.CombinedOutput()
	if err != nil {
		err = errors.New("can't run ffprobe: " + err.Error())
		return
	}
	var fileFormat FileFormat
	err = json.Unmarshal(out, &fileFormat)
	if err != nil {
		err = errors.New("can't parse ffprobe output: " + err.Error())
		return
	}
	duration, _ := strconv.ParseFloat(fileFormat.Format.Duration, 64)
	if duration < 0.05 {
		log.Println(helpers.ToJSON(fileFormat))
		err = errors.New("video size < 0.05 seconds")
		return
	}
	if duration < 20 {
		maxAmount = 1
	}
	var startOffset float64
	if randomize {
		startOffset = rand.Float64() * math.Min(duration*0.1, interval*0.8)
	}
	if maxAmount > 0 && interval > 0 {
		if (duration-startOffset)/(float64(maxAmount+1)) < interval {
			maxAmount = 0
		} else {
			interval = 0
		}
	}
	var start float64
	if maxAmount == 0 {
		maxAmount = int64(math.Floor((duration - startOffset) / interval))
		if maxAmount < 1 {
			maxAmount = 1
			start = (duration - startOffset) / 2
		} else {
			start = (duration - startOffset - interval*float64(maxAmount)) / 2
		}
	} else if interval == 0 {
		interval = (duration - startOffset) / float64(maxAmount+1)
		start = interval
	}
	var i int64
	for i = 0; i < maxAmount; i++ {
		seek := time.Duration(start+startOffset+float64(i)*interval) * time.Second
		resultFileName := "frame." + strconv.FormatInt(i, 10) + ".png"
		frameFile := filepath.Join(destinationPath, resultFileName)
		err = ExtractFrame(file, seek, frameFile)
		if err != nil {
			return
		}
	}
	return
}

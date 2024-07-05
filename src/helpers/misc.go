package helpers

import (
	"encoding/json"
	"image"
	"os"
	"strconv"
	"time"

	"github.com/agext/regexp"
)



func ToJSON(doc interface{}) string {
	bt, err := json.Marshal(doc)
	if err != nil {
		return ""
	}
	return string(bt)
}


func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}


func ParseHumanDuration(human string) time.Duration {
	var hours float64 = 0
	var minutes float64 = 0
	var seconds float64 = 0
	res := regexp.MustCompile(`([\d.]+)\s*y`).FindStringSubmatch(human)
	if res != nil {
		if i, err := strconv.ParseFloat(res[1], 64); err == nil {
			hours += i * 8760
		}
	}
	monthPresent := false
	res = regexp.MustCompile(`([\d.]+)\s*mo`).FindStringSubmatch(human)
	if res != nil {
		if i, err := strconv.ParseFloat(res[1], 64); err == nil {
			hours += i * 730
		}
		monthPresent = true
	}
	res = regexp.MustCompile(`([\d.]+)\s*d`).FindStringSubmatch(human)
	if res != nil {
		if i, err := strconv.ParseFloat(res[1], 64); err == nil {
			hours += i * 24
		}
	}
	res = regexp.MustCompile(`([\d.]+)\s*h`).FindStringSubmatch(human)
	if res != nil {
		if i, err := strconv.ParseFloat(res[1], 64); err == nil {
			hours += i
		}
	}
	if !monthPresent {
		res = regexp.MustCompile(`([\d.]+)\s*m`).FindStringSubmatch(human)
	} else {
		res = regexp.MustCompile(`([\d.]+)\s*mi`).FindStringSubmatch(human)
	}
	if res != nil {
		if i, err := strconv.ParseFloat(res[1], 64); err == nil {
			minutes += i
		}
	}
	res = regexp.MustCompile(`([\d.]+)\s*(s|$)`).FindStringSubmatch(human)
	if res != nil {
		if i, err := strconv.ParseFloat(res[1], 64); err == nil {
			seconds += i
		}
	}
	return time.Duration(hours*float64(time.Hour)) + time.Duration(minutes*float64(time.Minute)) + time.Duration(seconds*float64(time.Second))
}

func GetImageDimensions(imagePath string) (width int, height int, err error) {
	var file *os.File
	file, err = os.Open(imagePath)
	if err != nil {
		return
	}
	defer file.Close()
	var im image.Config
	im, _, err = image.DecodeConfig(file)
	if err != nil {
		return
	}
	width = im.Width
	height = im.Height
	return
}

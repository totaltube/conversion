package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "image/png"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/totaltube/conversion/helpers"
	"github.com/totaltube/conversion/types"
)

func ConvertVideoOld(files []string, resultFile string, format VideoFormat) error {
	var sourceFile string
	guid := xid.New()
	tmpdir := filepath.Join(conversionPath, "tmp", guid.String())
	err := os.MkdirAll(tmpdir, 0755)
	if err != nil {
		return errors.New("can't create temporary directory " + tmpdir + ": " + err.Error())
	}
	defer os.RemoveAll(tmpdir)
	if len(files) == 0 {
		return errors.New("file/files for video conversion not provided")
	}
	if len(files) > 1 {
		// Need to concatenate videos first
		fileExtension := ""
		concatFile := ""
		for _, f := range files {
			ext := strings.ToLower(filepath.Ext(f))
			if fileExtension == "" {
				fileExtension = ext
			} else if fileExtension != ext {
				return errors.New("can't concatenate files with different extensions ( " + fileExtension + " and " + ext + " )")
			}
			concatFile += "file " + filepath.Clean(f) + "\n"
		}
		concatFilePath := filepath.Join(tmpdir, "concat.txt")
		err = ioutil.WriteFile(concatFilePath, []byte(concatFile), 0644)
		if err != nil {
			return errors.New("can't write " + concatFilePath + " file: " + err.Error())
		}
		sourceFile = filepath.Join(tmpdir, "__file."+fileExtension)
		command := exec.Command("ffmpeg", "-y", "-f", "concat", "-i", concatFilePath, "-c", "copy", sourceFile)
		out, err := command.CombinedOutput()
		if err != nil {
			log.Println(string(out))
			return errors.New("can't concatenate video files: " + err.Error())
		}
	} else {
		sourceFile = files[0]
	}
	tempPath := filepath.Join(tmpdir, "temp")
	_ = os.MkdirAll(tempPath, 0755)
	cmd := format.Command
	cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE%", sourceFile)
	cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE_BASE%", filepath.Base(sourceFile))
	cmd = strings.ReplaceAll(cmd, "%WORKING_PATH%", tmpdir)
	cmd = strings.ReplaceAll(cmd, "%RESULT_FILE%", filepath.Base(resultFile))
	cmd = strings.ReplaceAll(cmd, "%RESULT_FILE_BASE%", filepath.Base(resultFile))
	cmd = strings.ReplaceAll(cmd, "%SOURCE_EXTENSION%", filepath.Ext(sourceFile))
	cmd = strings.ReplaceAll(cmd, "%VIDEO_BITRATE%", strconv.FormatUint(format.VideoBitrate, 10))
	cmd = strings.ReplaceAll(cmd, "%AUDIO_BITRATE%", strconv.FormatUint(format.AudioBitrate, 10))
	cmd = strings.ReplaceAll(cmd, "%RESIZE_OPTIONS%", format.ResizeOptions)
	cmd = strings.ReplaceAll(cmd, "%TEMP_PATH%", tempPath)
	cmd = strings.ReplaceAll(cmd, "%FFMPEG%", "ffmpeg")
	cmds := strings.Split(cmd, " && ")
	for _, cm := range cmds {
		nameArgs := strings.Split(cm, " ")
		name := nameArgs[0]
		args := nameArgs[1:]
		var argsFiltered []string
		for _, arg := range args {
			if arg != "" {
				argsFiltered = append(argsFiltered, arg)
			}
		}
		command := exec.Command(name, argsFiltered...)
		command.Dir = filepath.Dir(resultFile)
		out, err := command.CombinedOutput()
		if err != nil {
			log.Println(string(out))
			return errors.New("can't run command " + name + " " + strings.Join(argsFiltered, " ") + ": " + err.Error())
		}
	}
	if !helpers.FileExists(resultFile) {
		return errors.New("result file " + resultFile + " not created during conversion. Something is wrong.")
	}
	return nil
}

func ConvertVideoAction(sourceFile, resultFile string, videoBitrate, audioBitrate uint64) error {
	sourceFile, _ = filepath.Abs(sourceFile)
	resultFile, _ = filepath.Abs(resultFile)
	cmd := exec.Command("ffprobe", sourceFile, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", "-print_format", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("can't run ffprobe: " + err.Error())
	}
	var fileFormat FileFormat
	err = json.Unmarshal(out, &fileFormat)
	if err != nil {
		return errors.New("can't parse ffprobe output: " + err.Error())
	}
	for _, stream := range fileFormat.Streams {
		if stream.CodecType == "audio" {
			sourceAudioBitrate, _ := strconv.ParseUint(stream.BitRate, 10, 64)
			sourceAudioBitrate = sourceAudioBitrate / 1000
			if sourceAudioBitrate != 0 && (sourceAudioBitrate < audioBitrate || audioBitrate == 0) {
				audioBitrate = sourceAudioBitrate
			}
		}
		if stream.CodecType == "video" {
			sourceVideoBitrate, _ := strconv.ParseUint(stream.BitRate, 10, 64)
			sourceVideoBitrate = sourceVideoBitrate / 1000
			if sourceVideoBitrate != 0 && (sourceVideoBitrate < videoBitrate || videoBitrate == 0) {
				videoBitrate = sourceVideoBitrate
			}
		}
	}
	format := VideoFormat{VideoBitrate: videoBitrate, AudioBitrate: audioBitrate, ResizeOptions: ""}
	format.Command = `%FFMPEG% -y -i %SOURCE_FILE% -an -pass 1 -vcodec libx264 %RESIZE_OPTIONS% -b:v %VIDEO_BITRATE%k -preset fast -threads 2 %RESULT_FILE% && %FFMPEG% -y -i %SOURCE_FILE% -acodec aac -strict experimental -ab %AUDIO_BITRATE%k -pass 2 -vcodec libx264 %RESIZE_OPTIONS% -b:v %VIDEO_BITRATE%k -preset fast -threads 2 -movflags faststart %RESULT_FILE%`

	return ConvertVideoOld([]string{sourceFile}, resultFile, format)
}

func ConvertVideo(sourceFiles []string, tempPath string, targetPath string, format types.VideoFormatShort) (info types.ContentVideoInfo, err error) {
	var sourceFile string
	if tempPath == "" {
		tempPath, err = os.MkdirTemp(conversionPath, "convert_video_")
		if err != nil {
			log.Println(err)
			return
		}
		defer func() {
			if err != nil {
				log.Println(tempPath)
			} else {
				os.RemoveAll(tempPath)
			}
		}()
	}
	if len(sourceFiles) > 1 {
		// Need to concatenate videos first
		fileExtension := ""
		concatFile := ""
		for _, f := range sourceFiles {
			ext := strings.ToLower(filepath.Ext(f))
			if fileExtension == "" {
				fileExtension = ext
			} else if fileExtension != ext {
				err = errors.New("can't concatenate files with different extensions ( " + fileExtension + " and " + ext + " )")
				return
			}
			absPath, _ := filepath.Abs(f)
			concatFile += "file " + absPath + "\n"
		}
		concatFilePath := filepath.Join(tempPath, "concat.txt")
		err = os.WriteFile(concatFilePath, []byte(concatFile), 0644)
		if err != nil {
			err = errors.New("can't write " + concatFilePath + " file: " + err.Error())
			return
		}
		sourceFile = filepath.Join(tempPath, "__file."+fileExtension)
		command := exec.Command("ffmpeg", "-y", "-f", "concat", "-i", concatFilePath, "-c", "copy", sourceFile)
		var out []byte
		out, err = command.CombinedOutput()
		if err != nil {
			log.Println(string(out))
			err = errors.New("can't concatenate video files " + helpers.ToJSON(sourceFiles) + ": " + err.Error())
			return
		}
	} else {
		sourceFile = sourceFiles[0]
	}
	sourceFile, _ = filepath.Abs(sourceFile)
	probeCmd := exec.Command("ffprobe", sourceFile, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams")
	var out []byte
	out, err = probeCmd.CombinedOutput()
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
	copyAudioStream := false
	copyVideoStream := false
	resizeOptions := ""
	formatVideoBitrate := format.VideoBitrate
	var sourceVideoBitrate uint64
	var sourceWidth int64
	var sourceHeight int64
	for _, stream := range fileFormat.Streams {
		if stream.CodecType == "audio" {
			sourceAudioBitrate, _ := strconv.ParseUint(stream.BitRate, 10, 64)
			sourceAudioBitrate = sourceAudioBitrate / 1000
			if sourceAudioBitrate != 0 && (sourceAudioBitrate < uint64(float32(format.AudioBitrate)*1.3) ||
				format.AudioBitrate == 0) {
				format.AudioBitrate = int32(sourceAudioBitrate)
				if format.Type == "mp4" && stream.CodecName == "aac" {
					copyAudioStream = true
				}
				if format.Type == "webm" && (stream.CodecName == "vorbis" || stream.CodecName == "opus") {
					copyAudioStream = true
				}
				if format.Type == "ogg" && stream.CodecName == "vorbis" {
					copyAudioStream = true
				}
			}
		}
		if stream.CodecType == "video" {
			sizeOk := true
			sourceWidth = int64(stream.Width)
			sourceHeight = int64(stream.Height)
			if format.Crop && (format.Size.Width != int64(stream.Width) || format.Size.Height != int64(stream.Height)) {
				sizeOk = false
			}
			if !format.Crop && math.Abs(float64(format.Size.Width-int64(stream.Width)))/float64(format.Size.Width) > 0.2 {
				sizeOk = false
			}
			if !format.Crop && math.Abs(float64(format.Size.Height-int64(stream.Height)))/float64(format.Size.Height) > 0.2 {
				sizeOk = false
			}
			sourceVideoBitrate, _ = strconv.ParseUint(stream.BitRate, 10, 64)
			sourceVideoBitrate = sourceVideoBitrate / 1000
			if sourceVideoBitrate != 0 && (sourceVideoBitrate < uint64(float32(format.VideoBitrate)*1.2) ||
				format.VideoBitrate == 0) {
				format.VideoBitrate = int32(sourceVideoBitrate)
				if format.Type == "mp4" && (stream.CodecName == "h264") {
					copyVideoStream = true
				}
				if format.Type == "webm" && (stream.CodecName == "vp8" || stream.CodecName == "vp9" || stream.CodecName == "h264") {
					copyVideoStream = true
				}
				if format.Type == "ogg" && stream.CodecName == "theora" {
					copyVideoStream = true
				}
			}
			if !sizeOk {
				if format.Crop {
					resizeOptions = fmt.Sprintf(`-vf scale=(iw*sar)*max(%d/(iw*sar)\,%d/ih):ih*max(%d/(iw*sar)\,%d/ih),crop=%d:%d`,
						format.Size.Width, format.Size.Height, format.Size.Width, format.Size.Height, format.Size.Width, format.Size.Height)
				} else {
					if float64(format.Size.Width)/float64(format.Size.Height) > float64(stream.Width)/float64(stream.Height) {
						resizeOptions = fmt.Sprintf(`-vf scale=-2:%d`, format.Size.Height)
					} else {
						resizeOptions = fmt.Sprintf(`-vf scale=%d:-2`, format.Size.Width)
					}
				}
				copyVideoStream = false
			}
		}
	}
	if float64(sourceVideoBitrate) < float64(formatVideoBitrate)*1.2 && (float64(sourceWidth) < float64(format.Size.Width)*0.8 || float64(sourceHeight) < float64(format.Size.Height)*0.8) {
		err = errors.New("source file is too low quality to create this format")
		return
	}

	cmd := format.Command
	if cmd == "" {
		cmd = "ffmpeg -y -i %SOURCE_FILE% -an -pass 1 %VIDEO_OPTIONS% %RESIZE_OPTIONS% -preset fast -threads 2 -f mp4 /dev/null && ffmpeg -y -i %SOURCE_FILE% %AUDIO_OPTIONS% -pass 2 %VIDEO_OPTIONS% %RESIZE_OPTIONS% -preset fast -threads 2 "
		if format.Type == "mp4" {
			cmd += "-movflags faststart "
		}
		cmd += "%RESULT_FILE%"
	}
	if copyAudioStream {
		cmd = strings.ReplaceAll(cmd, "%AUDIO_OPTIONS%", "-c:a copy")
	} else {
		cmd = strings.ReplaceAll(cmd, "%AUDIO_OPTIONS%", fmt.Sprintf("-b:a %dk", format.AudioBitrate))
	}
	if copyVideoStream {
		cmd = strings.ReplaceAll(cmd, "%VIDEO_OPTIONS%", "-c:v copy")
	} else {
		cmd = strings.ReplaceAll(cmd, "%VIDEO_OPTIONS%", fmt.Sprintf("-b:v %dk", format.VideoBitrate))
	}
	var resultFile = filepath.Join(targetPath, fmt.Sprintf("video-%s.%s", format.Name, format.Type))
	resultFileAbs, _ := filepath.Abs(resultFile)
	syscall.Sync()
	cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE%", sourceFile)
	cmd = strings.ReplaceAll(cmd, "%SOURCE_FILE_BASE%", filepath.Base(sourceFile))
	cmd = strings.ReplaceAll(cmd, "%WORKING_PATH%", tempPath)
	cmd = strings.ReplaceAll(cmd, "%RESULT_FILE%", resultFileAbs)
	cmd = strings.ReplaceAll(cmd, "%RESULT_FILE_BASE%", filepath.Base(resultFileAbs))
	cmd = strings.ReplaceAll(cmd, "%SOURCE_EXTENSION%", filepath.Ext(sourceFile))
	cmd = strings.ReplaceAll(cmd, "%VIDEO_BITRATE%", strconv.FormatUint(uint64(format.VideoBitrate), 10))
	cmd = strings.ReplaceAll(cmd, "%AUDIO_BITRATE%", strconv.FormatUint(uint64(format.AudioBitrate), 10))
	cmd = strings.ReplaceAll(cmd, "%RESIZE_OPTIONS%", resizeOptions)
	cmd = strings.ReplaceAll(cmd, "%TEMP_PATH%", "")
	cmd = strings.ReplaceAll(cmd, "%FFMPEG%", "ffmpeg")
	// Создаем команду для выполнения через bash
	command := exec.Command("bash")
	command.Dir, _ = filepath.Abs(tempPath)
	// Создаем буфер, куда будем записывать стандартный вывод команды
	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	command.Stdout = &outBuffer
	command.Stderr = &errBuffer
	command.Stdin = bytes.NewBufferString(cmd)
	// Запускаем команду
	err = command.Run()
	if err != nil {
		err = errors.Wrap(err, "can't run command for format "+format.Name)
		log.Println(errBuffer.String())
		log.Println(outBuffer.String())
		return
	}
	syscall.Sync()
	if !helpers.FileExists(resultFile) {
		err = errors.New("result file " + resultFile + " not created during conversion. Something is wrong.")
	}
	probeCmd = exec.Command("ffprobe", resultFile, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams")
	out, err = probeCmd.CombinedOutput()
	if err != nil {
		err = errors.New("can't run ffprobe: " + err.Error())
		return
	}
	err = json.Unmarshal(out, &fileFormat)
	if err != nil {
		return
	}
	for _, v := range fileFormat.Streams {
		if v.CodecType == "video" {
			info.Size.Width = int64(v.Width)
			info.Size.Height = int64(v.Height)
			bitrate, _ := strconv.ParseInt(v.BitRate, 10, 32)
			info.VideoBitrate = int32(bitrate)
			info.Type = format.Type
			info.Duration, _ = strconv.ParseFloat(v.Duration, 32)
		}
		if v.CodecType == "audio" {
			bitrate, _ := strconv.ParseInt(v.BitRate, 10, 32)
			info.AudioBitrate = int32(bitrate)
		}
	}
	if info.Duration <= 0 || info.Size.Width <= 0 || info.Size.Height <= 0 {
		err = errors.New("wrong target video format")
		return
	}
	if format.CreatePoster {
		var seekPercent = rand.Float64()*(format.PosterTimeRange[1]-format.PosterTimeRange[0]) + format.PosterTimeRange[0]
		posterFile := filepath.Join(targetPath, fmt.Sprintf("poster-%s.%s", format.Name, format.PosterType))
		tempFile := filepath.Join(tempPath, "poster.png")
		err = ExtractFrame(resultFile, time.Duration(float64(time.Second)*info.Duration*seekPercent/100), tempFile)
		if err != nil {
			log.Println(err)
			err = errors.Wrap(err, "can't extract poster from result video")
			return
		}
		if format.PosterType == "png" {
			err = os.Rename(tempFile, posterFile)
			if err != nil {
				log.Println(err)
				err = errors.Wrap(err, "poster create error")
				return
			}
		} else {
			if format.PosterCommand == "" {
				if format.PosterType == "webp" {
					format.PosterCommand = "convert %SOURCE_FILE% -thumbnail %SIZE%^ -gravity center -extent %SIZE% -quality 92 %RESULT_FILE%"
				} else {
					format.PosterCommand = "convert %SOURCE_FILE% -thumbnail %SIZE%^ -gravity center -extent %SIZE% -quality 84 %RESULT_FILE%"
				}
			}
			err = ConvertImage(tempFile, format.PosterCommand, posterFile, "")
			if err != nil {
				log.Println(err)
				err = errors.Wrap(err, "poster create error")
				return
			}
		}
		info.PosterType = format.PosterType
	}
	if format.CreateTimeline {
		_ = os.MkdirAll(filepath.Join(tempPath, "timeline"), os.ModePerm)
		err = doExtractFrames2(resultFile, filepath.Join(tempPath, "timeline"),
			int64(format.TimelineMaxAmount), float64(format.TimelineMinInterval), false)
		if err != nil {
			err = errors.Wrap(err, "timeline create error")
			log.Println(err)
			return
		}
		matches, _ := filepath.Glob(filepath.Join(tempPath, "timeline", "*.png"))
		if len(matches) > 0 {
			info.TimelineFrames = int32(len(matches))
			var gap = info.Duration / float64(len(matches))
			var cmd = "convert %SOURCE_FILE% -resize %SIZE% %RESULT_FILE%"
			if format.TimelineCrop {
				cmd = "convert %SOURCE_FILE% -thumbnail %SIZE%^ -gravity center -extent %SIZE% %RESULT_FILE%"
			}
			var currentWidth int
			var currentDuration float64
			var vttContents = `WEBVTT
			
`
			timelineFile := fmt.Sprintf("timeline-%s.%s", format.Name, format.TimelineType)
			var timelineImages = make([]string, 0, len(matches))
			for k, m := range matches {
				frameFile := fmt.Sprintf("%d.png", k)
				err = ConvertImage(m, cmd, filepath.Join(tempPath, "timeline", frameFile), fmt.Sprintf("%dx%d", format.TimelineSize.Width, format.TimelineSize.Height))
				_ = os.Remove(m)
				if err != nil {
					err = errors.Wrap(err, "timeline frame conversion error")
					log.Println(err)
					return
				}
				var width int
				var height int
				width, height, err = helpers.GetImageDimensions(filepath.Join(tempPath, "timeline", frameFile))
				if err != nil {
					err = errors.Wrap(err, "error getting dimensions of timeline frame")
					log.Println(err)
					return
				}
				info.TimelineSize.Width = int64(width)
				info.TimelineSize.Height = int64(height)
				z := time.Unix(0, 0).UTC()
				nextDuration := math.Min(info.Duration, gap+currentDuration)
				vttContents += fmt.Sprintf(`%s.000 --> %s.000
%s#xywh=%d,0,%d,%d

`,
					z.Add(time.Duration(currentDuration*float64(time.Second))).Format("15:04:05"),
					z.Add(time.Duration(nextDuration*float64(time.Second))).Format("15:04:05"),
					timelineFile, currentWidth, width, height)
				currentWidth += width
				currentDuration += gap
				timelineImages = append(timelineImages, filepath.Join(tempPath, "timeline", frameFile))
			}
			err = os.WriteFile(filepath.Join(targetPath, fmt.Sprintf("timeline-%s.vtt", format.Name)), []byte(vttContents), os.ModePerm)
			if err != nil {
				log.Println(err)
				err = errors.Wrap(err, "can't write vtt file")
				return
			}
			var args = []string{"+append"}
			for _, ti := range timelineImages {
				args = append(args, ti)
			}
			if format.TimelineType == "jpg" {
				args = append(args, "-quality", "85")
			} else if format.TimelineType == "webp" {
				args = append(args, "-quality", "92")
			}
			args = append(args, filepath.Join(targetPath, timelineFile))
			command := exec.Command("convert", args...)
			out, err = command.CombinedOutput()
			if err != nil {
				err = errors.Wrap(err, "error creating timeline combined image")
				log.Println(err, string(out))
				return
			}
			syscall.Sync()
			info.TimelineType = format.TimelineType
		}
	}
	return
}

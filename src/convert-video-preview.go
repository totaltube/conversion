package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/totaltube/conversion/types"
)

// ExtractVideoSegments извлекает несколько сегментов из видео файла
func ExtractVideoSegments(sourceFile string, tempPath string, segmentsCount int64, segmentDuration float64) ([]string, error) {
	// Получаем информацию о видео
	cmd := exec.Command("ffprobe", sourceFile, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "can't run ffprobe for video info")
	}

	var fileFormat types.FileFormat
	err = json.Unmarshal(out, &fileFormat)
	if err != nil {
		return nil, errors.Wrap(err, "can't parse ffprobe output")
	}

	duration, err := strconv.ParseFloat(fileFormat.Format.Duration, 64)
	if err != nil {
		return nil, errors.Wrap(err, "can't parse video duration")
	}

	if duration < segmentDuration*float64(segmentsCount) {
		return nil, errors.New("video duration too short for requested segments")
	}

	// Создаем директорию для сегментов
	segmentsDir := filepath.Join(tempPath, "video_segments")
	err = os.MkdirAll(segmentsDir, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "can't create segments directory")
	}

	var segmentFiles []string

	// Выбираем точки для извлечения сегментов
	// Распределяем равномерно по длительности видео
	interval := duration / float64(segmentsCount+1)

	for i := range segmentsCount {
		// Добавляем небольшую рандомизацию чтобы сегменты не были слишком предсказуемыми
		randomOffset := rand.Float64() * interval * 0.3 // ±15% от интервала
		seekTime := interval*float64(i+1) + randomOffset

		if seekTime+segmentDuration > duration {
			seekTime = duration - segmentDuration
		}
		if seekTime < 0 {
			seekTime = 0
		}

		segmentFile := filepath.Join(segmentsDir, fmt.Sprintf("segment_%d.mp4", i))
		segmentFiles = append(segmentFiles, segmentFile)

		// Извлекаем сегмент
		seekSeconds := strconv.FormatFloat(seekTime, 'f', 2, 64)
		durationStr := strconv.FormatFloat(segmentDuration, 'f', 2, 64)

		cmd := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
			"-ss", seekSeconds,
			"-i", sourceFile,
			"-t", durationStr,
			"-c", "copy", // копируем без перекодирования для скорости
			segmentFile)

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Error extracting segment %d: %s", i, string(out))
			return nil, errors.Wrapf(err, "can't extract video segment %d", i)
		}
	}

	return segmentFiles, nil
}

// ConcatVideoSegments склеивает несколько видео сегментов в один файл
func ConcatVideoSegments(segmentFiles []string, outputFile string) error {
	if len(segmentFiles) == 0 {
		return errors.New("no segment files provided")
	}

	if len(segmentFiles) == 1 {
		// Если только один сегмент, просто копируем
		return exec.Command("cp", segmentFiles[0], outputFile).Run()
	}

	// Создаем файл со списком для concat
	concatFile := outputFile + ".concat.txt"
	var concatContent strings.Builder

	for _, segment := range segmentFiles {
		concatContent.WriteString(fmt.Sprintf("file '%s'\n", segment))
	}

	err := os.WriteFile(concatFile, []byte(concatContent.String()), 0644)
	if err != nil {
		return errors.Wrap(err, "can't write concat file")
	}
	defer os.Remove(concatFile)

	// Склеиваем сегменты
	cmd := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
		"-c", "copy",
		outputFile)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error concatenating segments: %s", string(out))
		return errors.Wrap(err, "can't concatenate video segments")
	}

	return nil
}

// CreateVideoPreview создает video preview из исходного видео файла
func CreateVideoPreview(sourceFile string, tempPath string, format types.ThumbFormatShort, outputFile string) error {
	// Извлекаем сегменты
	segmentFiles, err := ExtractVideoSegments(sourceFile, tempPath, format.SegmentsCount, format.SegmentDuration)
	if err != nil {
		return errors.Wrap(err, "can't extract video segments")
	}

	// Склеиваем сегменты в preview
	concatFile := filepath.Join(tempPath, "preview_concat.mp4")
	err = ConcatVideoSegments(segmentFiles, concatFile)
	if err != nil {
		return errors.Wrap(err, "can't concatenate segments")
	}

	// Применяем финальное кодирование с нужными параметрами размера и битрейта
	sizeStr := fmt.Sprintf("%dx%d", format.VideoSize.Width, format.VideoSize.Height)
	bitrateStr := fmt.Sprintf("%dk", format.VideoBitrate)

	cmd := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", concatFile,
		"-vf", fmt.Sprintf("scale=%s", sizeStr),
		"-b:v", bitrateStr,
		"-r", "25",
		"-c:v", "libx264",
		"-preset", "fast",
		"-an",
		outputFile)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error creating final preview: %s", string(out))
		return errors.Wrap(err, "can't create final video preview")
	}

	return nil
}

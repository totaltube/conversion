package main

import (
	"github.com/gabriel-vasile/mimetype"
	"testing"
)

func TestMimeType(t *testing.T) {
	mime, err := mimetype.DetectFile("../test/video.mp4")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(mime.String())
}

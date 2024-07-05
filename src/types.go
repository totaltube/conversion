package main

type M map[string]interface{}

type AppendFormat struct {
	Quality int    `json:"quality"`
	Type    string `json:"type"`
}

type ExtractFramesFormat struct {
	TimeOffset float64 `json:"timeOffset"`
	Single     bool    `json:"single"`
	Start      float64 `json:"start"`
	Amount     int64   `json:"amount"`
	Interval   float64 `json:"interval"`
	Duration   float64 `json:"duration"`
}

type FileFormat struct {
	Streams []FStream `json:"streams"`
	Format  FFormat   `json:"format"`
}

type FFormat struct {
	Filename string `json:"filename"`
	Duration string `json:"duration"`
	Size     string `json:"size"`
	BitRate  string `json:"bit_rate"`
}

type FStream struct {
	CodecName          string `json:"codec_name"`
	CodecType          string `json:"codec_type"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	Duration           string `json:"duration"`
	BitRate            string `json:"bit_rate"`
	DisplayAspectRatio string `json:"display_aspect_ratio"`
}

type ImageFormat struct {
	Size    string `json:"size"`
	Command string `json:"command"`
	Type    string `json:"type"`
}

type TimelineFormat struct {
	Merge   bool   `json:"merge"`
	Type    string `json:"type"`
	Size    string `json:"size"`
	Command string `json:"command"`
}

type VideoFormat struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	VideoBitrate  uint64 `json:"videoBitrate"`
	AudioBitrate  uint64 `json:"audioBitrate"`
	ResizeOptions string `json:"resizeOptions"`
}
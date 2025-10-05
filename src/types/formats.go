package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
	"github.com/ysmood/gson"
)

type ContentTypes []ContentType

func (c *ContentTypes) UnmarshalJSON(b []byte) error {
	var ss []string
	err := json.Unmarshal(b, &ss)
	if err != nil {
		return err
	}
	var ct = make([]ContentType, len(ss))
	for _, s := range ss {
		switch s {
		case ContentTypeChannel.String():
			ct = append(ct, ContentTypeChannel)
		case ContentTypeModel.String():
			ct = append(ct, ContentTypeModel)
		case ContentTypeCategory.String():
			ct = append(ct, ContentTypeCategory)
		case ContentTypeVideoLink.String():
			ct = append(ct, ContentTypeVideoLink)
		case ContentTypeVideo.String():
			ct = append(ct, ContentTypeVideo)
		case ContentTypeGallery.String():
			ct = append(ct, ContentTypeGallery)
		case ContentTypeVideoEmbed.String():
			ct = append(ct, ContentTypeVideoEmbed)
		case ContentTypeStaticPage.String():
			ct = append(ct, ContentTypeStaticPage)
		}
	}
	*c = ct
	return nil
}

func (c ContentTypes) MarshalJSON() ([]byte, error) {
	var ss = make([]string, len(c))
	for _, ct := range c {
		ss = append(ss, ct.String())
	}
	return json.Marshal(ss)
}

type YesNo bool

func (c *YesNo) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	if s == "Y" {
		*c = true
	} else {
		*c = false
	}
	return nil
}
func (c YesNo) MarshalJSON() ([]byte, error) {
	if c {
		return json.Marshal("Y")
	} else {
		return json.Marshal("N")
	}
}

type Size struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

func (s Size) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%dx%d", s.Width, s.Height))
}
func (s *Size) UnmarshalJSON(b []byte) error {
	var ss string
	err := json.Unmarshal(b, &ss)
	if err != nil {
		return err
	}
	sizes := strings.Split(ss, "x")
	if len(sizes) != 2 {
		return errors.New("wrong size")
	}
	*s = Size{}
	s.Width, err = strconv.ParseInt(sizes[0], 10, 16)
	if err != nil {
		return err
	}
	s.Height, err = strconv.ParseInt(sizes[1], 10, 16)
	if err != nil {
		return err
	}
	return nil
}

type ThumbFormat struct {
	Name                string        `json:"name"`
	Sites               []int64       `json:"sites"`
	SiteGroups          []string      `json:"site_groups"`
	Types               []ContentType `json:"types"`
	Required            bool          `json:"required"`
	Size                Size          `json:"size"`
	MinSourceSize       Size          `json:"min_source_size"`
	MinSize             int64         `json:"min_size"`
	Command             string        `json:"command"`
	MinTimeInterval     float64       `json:"min_time_interval"`
	MaxThumbs           int64         `json:"max_thumbs"`
	Type                string        `json:"type"`
	Retina              bool          `json:"retina"`
	RetinaMinSourceSize Size          `json:"retina_min_source_size"`
	RetinaCommand       string        `json:"retina_command"`
}

// ThumbFormatShort for conversion server
type ThumbFormatShort struct {
	Name                string   `json:"name"`
	Command             string   `json:"command"`
	Size                Size     `json:"size"`
	MinSourceSize       Size     `json:"min_source_size"`
	MinSize             int64    `json:"min_size"`
	MinTimeInterval     float64  `json:"min_time_interval"`
	MaxThumbs           int64    `json:"max_thumbs"`
	Type                string   `json:"type"`
	Retina              bool     `json:"retina"`
	RetinaMinSourceSize Size     `json:"retina_min_source_size"`
	CreateVideoPreview  bool     `json:"create_video_preview"`
	VideoFormats        []string `json:"video_formats"`
	VideoSize           Size     `json:"video_size"`
	SegmentsCount       int64    `json:"segments_count"`
	SegmentDuration     float64  `json:"segment_duration"`
	VideoBitrate        int64    `json:"video_bitrate"`
}

func (tf ThumbFormat) CompatMarshalJSON() ([]byte, error) {
	var required = "N"
	if tf.Required {
		required = "Y"
	}
	var siteGroups = tf.SiteGroups
	var types = make([]string, 0, len(tf.Types))
	for _, t := range tf.Types {
		types = append(types, t.String())
	}
	var sites = make([]string, 0, len(tf.Sites))
	for _, s := range tf.Sites {
		sites = append(sites, strconv.FormatInt(s, 10))
	}

	var res = map[string]interface{}{
		"name":            tf.Name,
		"types":           types,
		"sites":           sites,
		"site_groups":     siteGroups,
		"required":        required,
		"size":            tf.Size,
		"command":         tf.Command,
		"minSourceSize":   tf.MinSourceSize,
		"minSize":         strconv.FormatInt(tf.MinSize, 10),
		"maxThumbs":       strconv.FormatInt(tf.MaxThumbs, 10),
		"type":            tf.Type,
		"minTimeInterval": fmt.Sprintf("%.2f", tf.MinTimeInterval),
	}
	if tf.Retina {
		res["retina"] = map[string]interface{}{
			"minSourceSize": tf.RetinaMinSourceSize,
			"command":       tf.RetinaCommand,
		}
	}
	return json.Marshal(res)
}
func (tf *ThumbFormat) CompatUnmarshalJSON(b []byte) error {
	var data gson.JSON
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	*tf = ThumbFormat{
		Name:    data.Get("name").String(),
		Command: data.Get("command").String(),
		Type:    data.Get("type").String(),
	}
	tf.RetinaCommand = data.Get("retina.command").String()
	if tf.RetinaCommand != "" {
		tf.Retina = true
		var size Size
		bt, _ := json.Marshal(data.Get("retina.minSourceSize").String())
		err = json.Unmarshal(bt, &size)
		if err != nil {
			return err
		}
		tf.RetinaMinSourceSize = size
	}
	siteGroups := data.Get("site_groups").Arr()
	tf.SiteGroups = make([]string, 0, len(siteGroups))
	for _, s := range siteGroups {
		val := s.String()
		if val == "" {
			continue
		}
		tf.SiteGroups = append(tf.SiteGroups, val)
	}
	sites := data.Get("sites").Arr()
	tf.Sites = make([]int64, 0, len(sites))
	for _, s := range sites {
		val := s.String()
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			tf.Sites = append(tf.Sites, id)
		} else {
			tf.SiteGroups = append(tf.SiteGroups, val)
		}
	}
	if data.Get("required").String() == "Y" {
		tf.Required = true
	} else if data.Get("required").String() == "N" {
		tf.Required = false
	} else {
		tf.Required = data.Get("required").Bool()
	}
	bt, _ := json.Marshal(data.Get("size").String())
	err = json.Unmarshal(bt, &tf.Size)
	if err != nil {
		return err
	}
	bt, _ = json.Marshal(data.Get("minSourceSize").String())
	err = json.Unmarshal(bt, &tf.MinSourceSize)
	if err != nil {
		return err
	}
	tf.MinSize, err = strconv.ParseInt(data.Get("minSize").String(), 10, 64)
	if err != nil {
		return err
	}
	if data.Get("minTimeInterval").Num() > 0 {
		tf.MinTimeInterval = data.Get("minTimeInterval").Num()
	} else {
		tf.MinTimeInterval, _ = strconv.ParseFloat(data.Get("minTimeInterval").String(), 64)
	}
	tf.MaxThumbs, _ = strconv.ParseInt(data.Get("maxThumbs").String(), 10, 64)
	gTypes := data.Get("types").Arr()
	var types = make([]ContentType, 0, len(gTypes))
	for _, t := range gTypes {
		if t.Int() != 0 {
			types = append(types, ContentType(t.Int()))
		} else {
			for _, tt := range []ContentType{1, 2, 3, 4, 5, 6, 7, 8, 9} {
				if tt.String() == t.String() {
					types = append(types, tt)
				}
			}
		}
	}
	tf.Types = types
	return nil
}

var sizeValidation = validation.By(func(value interface{}) error {
	if size, ok := value.(Size); ok {
		if size.Width <= 0 || size.Width >= 100500 || size.Height <= 0 || size.Height >= 100500 {
			return errors.New("wrong size format")
		}
	} else if size, ok := value.(*Size); ok && size != nil {
		if size.Width <= 0 || size.Width >= 100500 || size.Height <= 0 || size.Height >= 100500 {
			return errors.New("wrong size format")
		}
	}
	return nil
})

func (tf ThumbFormat) Validate() error {
	return validation.ValidateStruct(&tf,
		validation.Field(&tf.Name, validation.Required),
		validation.Field(&tf.Size, validation.Required, sizeValidation),
		validation.Field(&tf.MinSourceSize, validation.Required, sizeValidation),
		validation.Field(&tf.MinSize, validation.Required, validation.Min(0)),
		validation.Field(&tf.Command, validation.Required),
		validation.Field(&tf.MinTimeInterval, validation.Required),
		validation.Field(&tf.MaxThumbs, validation.Required, validation.Min(1), validation.Max(100)),
		validation.Field(&tf.Type, validation.Required, validation.In("jpg", "webp", "png")),
		validation.Field(&tf.RetinaCommand, validation.When(tf.Retina, validation.Required)),
		validation.Field(&tf.RetinaMinSourceSize, validation.When(tf.Retina, validation.Required, sizeValidation)),
	)
}

type GalleryFormat struct {
	Name            string   `json:"name"`
	Sites           []int64  `json:"sites"`
	SiteGroups      []string `json:"site_groups"`
	Required        bool     `json:"required"`
	Size            Size     `json:"size"`
	PreviewSize     Size     `json:"preview_size"`
	MinSourceSize   Size     `json:"min_source_size"`
	Command         string   `json:"command"`
	MaxAmount       int64    `json:"max_amount"`
	MinAmount       int64    `json:"min_amount"`
	MinTimeInterval float64  `json:"min_time_interval"`
	Type            string   `json:"type"`
}

func (f GalleryFormat) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Name, validation.Required),
		validation.Field(&f.Size, validation.Required, sizeValidation),
		validation.Field(&f.PreviewSize, validation.Required, sizeValidation),
		validation.Field(&f.MinSourceSize, validation.Required, sizeValidation),
		validation.Field(&f.Type, validation.Required, validation.In("jpg", "webp", "png")),
	)
}

type GalleryFormatShort struct {
	Name            string  `json:"name"`
	Command         string  `json:"command"`
	Size            Size    `json:"size"`
	PreviewSize     Size    `json:"preview_size"`
	MinSourceSize   Size    `json:"min_source_size"`
	MinTimeInterval float64 `json:"min_time_interval"`
	MaxAmount       int64   `json:"max_amount"`
	MinAmount       int64   `json:"min_amount"`
	Type            string  `json:"type"`
}

type VideoFormat struct {
	Name                string     `json:"name"`
	Sites               []int64    `json:"sites"`
	SiteGroups          []string   `json:"site_groups"`
	Required            bool       `json:"required"`
	Size                Size       `json:"size"`
	Crop                bool       `json:"crop"`
	VideoBitrate        int32      `json:"video_bitrate"`
	AudioBitrate        int32      `json:"audio_bitrate"`
	Command             string     `json:"command"`
	Type                string     `json:"type"`
	CreatePoster        bool       `json:"create_poster"`
	PosterType          string     `json:"poster_type"`
	PosterTimeRange     [2]float64 `json:"poster_time_range"`
	PosterCommand       string     `json:"poster_command"`
	CreateTimeline      bool       `json:"create_timeline"`
	TimelineSize        Size       `json:"timeline_size"`
	TimelineCrop        bool       `json:"timeline_crop"`
	TimelineMaxAmount   int32      `json:"timeline_max_amount"`
	TimelineMinInterval float32    `json:"timeline_min_interval"`
	TimelineType        string     `json:"timeline_type"`
}

type VideoFormatShort struct {
	Name                string     `json:"name"`
	Size                Size       `json:"size"`
	Crop                bool       `json:"crop"`
	VideoBitrate        int32      `json:"video_bitrate"`
	AudioBitrate        int32      `json:"audio_bitrate"`
	Command             string     `json:"command"`
	Type                string     `json:"type"`
	CreatePoster        bool       `json:"create_poster"`
	PosterType          string     `json:"poster_type"`
	PosterTimeRange     [2]float64 `json:"poster_time_range"`
	PosterCommand       string     `json:"poster_command"`
	CreateTimeline      bool       `json:"create_timeline"`
	TimelineSize        Size       `json:"timeline_size"`
	TimelineCrop        bool       `json:"timeline_crop"`
	TimelineMaxAmount   int32      `json:"timeline_max_amount"`
	TimelineMinInterval float32    `json:"timeline_min_interval"`
	TimelineType        string     `json:"timeline_type"`
}

func (f VideoFormat) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Name, validation.Required),
		validation.Field(&f.Size, validation.Required, sizeValidation),
		validation.Field(&f.VideoBitrate, validation.Min(int32(0))),
		validation.Field(&f.AudioBitrate, validation.Min(int32(0))),
		validation.Field(&f.Type, validation.Required),
		validation.Field(&f.PosterType, validation.When(f.CreatePoster, validation.Required, validation.In("jpg", "webp", "png"))),
		validation.Field(&f.TimelineSize, validation.When(f.CreateTimeline, validation.Required, sizeValidation)),
		validation.Field(&f.TimelineMaxAmount, validation.When(f.CreateTimeline, validation.Min(int32(0)), validation.Max(int32(500)))),
		validation.Field(&f.TimelineMinInterval, validation.When(f.CreateTimeline, validation.Min(float32(0)))),
		validation.Field(&f.TimelineType, validation.When(f.CreateTimeline, validation.Required), validation.In("jpg", "webp", "png")),
	)
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

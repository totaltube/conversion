package types



type ContentType int64

const (
	ContentTypeStaticPage   ContentType = 1
	ContentTypeVideoEmbed   ContentType = 2
	ContentTypeVideoLink    ContentType = 3
	ContentTypeVideo        ContentType = 4
	ContentTypeGallery      ContentType = 5
	ContentTypeLink         ContentType = 6
	ContentTypeCategory     ContentType = 7
	ContentTypeChannel      ContentType = 8
	ContentTypeModel        ContentType = 9
	ContentTypeVideoHotlink ContentType = 10
)

func ContentTypeFromString(str string) ContentType {
	for _, t := range []ContentType{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if t.String() == str {
			return t
		}
	}
	return 0
}

func (c ContentType) String() string {
	switch c {
	case ContentTypeStaticPage:
		return "static-page"
	case ContentTypeVideoEmbed:
		return "video-embed"
	case ContentTypeVideoLink:
		return "video-link"
	case ContentTypeVideoHotlink:
		return "video-hotlink"
	case ContentTypeVideo:
		return "video"
	case ContentTypeGallery:
		return "gallery"
	case ContentTypeLink:
		return "link"
	case ContentTypeCategory:
		return "category"
	case ContentTypeChannel:
		return "channel"
	case ContentTypeModel:
		return "model"
	}
	return "unknown"
}

type ContentVideoInfo struct {
	Type           string  `json:"type"`
	Size           Size    `json:"size"`
	VideoBitrate   int32   `json:"video_bitrate"`
	AudioBitrate   int32   `json:"audio_bitrate"`
	PosterType     string  `json:"poster_type,omitempty"`
	TimelineType   string  `json:"timeline_type,omitempty"`
	TimelineSize   Size    `json:"timeline_size"`
	TimelineFrames int32   `json:"timeline_frames"`
	Duration       float64 `json:"duration"`
}
package types

type MakeThumbsRequest struct {
	Source                  string           `json:"source"`
	VideoSource             string           `json:"video_source"`
	Destination             string           `json:"destination"`
	DestinationVideoPreview string           `json:"destination_video_preview"`
	MaxThumbs               int64            `json:"max_thumbs"`
	Format                  ThumbFormatShort `json:"format"`
}

type MakeImagesRequest struct {
	Source      string             `json:"source"`
	Destination string             `json:"destination"`
	Format      GalleryFormatShort `json:"format"`
}

type MakeVideoRequest struct {
	Source      string           `json:"source"`
	Destination string           `json:"destination"`
	Format      VideoFormatShort `json:"format"`
}

type VideoInfoRequest struct {
	Source string `json:"source"`
}

package types

type MakeThumbsRequest struct {
	Source      string           `json:"source"`
	Destination string           `json:"destination"`
	MaxThumbs   int64            `json:"max_thumbs"`
	Format      ThumbFormatShort `json:"format"`
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

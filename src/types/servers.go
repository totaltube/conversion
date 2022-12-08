package types

import (
	"log"
	"math/rand"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/agext/regexp"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/totaltube/conversion/helpers"
)

type StorageServers map[string]StorageServer
type StoreType string

const (
	StoreTypeThumbs            StoreType = "thumbs"
	StoreTypeOriginalImages    StoreType = "original_images"
	StoreTypePhotoAlbums       StoreType = "photo_albums"
	StoreTypeVideos            StoreType = "videos"
	StoreTypeOriginalVideos    StoreType = "original_videos"
	StoreTypeTimeline          StoreType = "timeline"
	StoreTypeTimelineOriginals StoreType = "timeline_originals"
	StoreTypeSources           StoreType = "sources"
	StoreTypeOther             StoreType = "other"
)

type StorageServer struct {
	Name     string            `json:"name"`
	ApiUrls  []string          `json:"apiUrls,omitempty"`
	ApiKey   string            `json:"apiKey,omitempty"`
	Endpoint string            `json:"endpoint"`
	Secure   bool              `json:"secure"`
	User     string            `json:"user"`
	Password string            `json:"password"`
	Bucket   string            `json:"bucket"`
	Url      string            `json:"url"`
	SiteUrls map[string]string `json:"siteUrls,omitempty"`
	Types    []StoreType       `json:"types,omitempty"`
	Weight   string            `json:"weight,omitempty"`
	Naming   string            `json:"naming"`
}

func (c StorageServers) Validate() error {
	for _, cc := range c {
		err := validation.ValidateStruct(&cc,
			validation.Field(&cc.Name, validation.Required, validation.Length(0, 50)),
			validation.Field(&cc.Url, validation.Required, validation.Length(0, 255), is.URL),
			validation.Field(&cc.Endpoint, validation.Required),
			validation.Field(&cc.User, validation.Required),
			validation.Field(&cc.Password, validation.Required),
			validation.Field(&cc.Bucket, validation.Required),
			validation.Field(&cc.SiteUrls, validation.Each(validation.Required,
				validation.Length(0, 255), is.URL)),
			validation.Field(&cc.Naming, validation.Required),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c StorageServer) S3() (endpoint string, secure bool, accessKey string, secretKey string, bucket string) {
	return c.Endpoint, c.Secure, c.User, c.Password, c.Bucket
}

func (c StorageServer) ToURL(Path string) string {
	var scheme = "https"
	if !c.Secure {
		scheme = "http"
	}
	user := url.UserPassword("s3!"+c.User, c.Password)
	var u = &url.URL{
		Scheme: scheme,
		User:   user,
		Host:   c.Endpoint,
		Path:   path.Join(c.Bucket, Path),
	}
	return u.String()
}

type ConversionServer struct {
	Name   string `json:"name"`
	ApiUrl string `json:"apiUrl"`
	ApiKey string `json:"apiKey"`
	Weight string `json:"weight"`
}

func (c ConversionServer) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Name, validation.Required, validation.Length(1, 50)),
		validation.Field(&c.ApiUrl, validation.Required, validation.Length(0, 255), is.URL),
		validation.Field(&c.ApiKey, validation.Required, validation.Length(12, 100)),
	)
}

func (c ConversionServers) Validate() error {
	for _, cc := range c {
		if err := cc.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type ConversionServers map[string]ConversionServer

func (c ConversionServers) GetOne() (result *ConversionServer, err error) {
	if len(c) == 0 {
		err = errors.New("no conversion servers defined")
		return
	}
	var fullWeight float64
	for _, cs := range c {
		weight, _ := strconv.ParseFloat(cs.Weight, 64)
		fullWeight += weight
	}
	var rnd = rand.Float64() * fullWeight
	var curWeight float64
	var lastKey string
	for k, cs := range c {
		weight, _ := strconv.ParseFloat(cs.Weight, 64)
		curWeight += weight
		if rnd < curWeight {
			cs := cs
			result = &cs
			return
		}
		lastKey = k
	}
	cs, _ := c[lastKey]
	result = &cs
	return
}
func (c StorageServers) GetByName(name string) (result *StorageServer) {
	for _, s := range c {
		if s.Name == name {
			result = &s
			return
		}
	}
	return
}
func (c StorageServers) GetType(tp StoreType) (result *StorageServer, err error) {
	var filtered = make([]StorageServer, 0, len(c))
	for _, s := range c {
		if lo.Contains(s.Types, tp) {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil, errors.New("no storage server of type " + string(tp) + " found")
	}
	var fullWeight float64
	for _, s := range filtered {
		var weight float64
		weight, err = strconv.ParseFloat(s.Weight, 64)
		if err != nil {
			log.Println(err)
			return
		}
		fullWeight += weight
	}
	var rnd = rand.Float64() * fullWeight
	var curWeight float64
	for _, s := range filtered {
		weight, _ := strconv.ParseFloat(s.Weight, 64)
		curWeight += weight
		if rnd < curWeight {
			s := s
			result = &s
			return
		}
	}
	result = &filtered[len(filtered)-1]
	return
}

// language=GoRegExp
var thumbsPathRegex = regexp.MustCompile(`%(slug|id|year|month|day|hash)%(\[(-?[\d]+),([\d]+)]|)`)

func (c StorageServer) GetContentPath(id int64, slug string, createdAt time.Time, subPath string) string {
	return path.Join(thumbsPathRegex.ReplaceAllStringSubmatchFunc(c.Naming, func(matched []string) string {
		var res string
		switch matched[1] {
		case "slug":
			res = slug
		case "id":
			res = strconv.FormatInt(id, 10)
		case "year":
			res = createdAt.Format("2006")
		case "month":
			res = createdAt.Format("1")
		case "day":
			res = createdAt.Format("2")
		case "hash":
			res = helpers.Md5Hash(slug)
		}
		if matched[2] != "" {
			from, _ := strconv.ParseInt(matched[3], 10, 64)
			to, _ := strconv.ParseInt(matched[4], 10, 64)
			rres := []rune(res)
			if from < 0 {
				from = int64(len(rres)) + from
				if from < 0 {
					from = 0
				}
			}
			to = from + to
			if to > int64(len(rres)) {
				to = int64(len(rres))
			}
			if to == from {
				rres = []rune{}
			} else {
				rres = rres[from:to]
			}
			res = string(rres)
		}
		return res
	}), subPath)
}

package types

import (
	"github.com/pkg/errors"
	"net/url"
	"path"
	"strings"
)

type S3ServerInterface interface {
	S3() (endpoint string, secure bool, accessKey string, secretKey string, bucket string)
	ToURL(Path string) string
}

// S3Server Generic S3 server struct
type S3Server struct {
	Endpoint   string `json:"endpoint"`
	Secure     bool   `json:"secure"`
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
	Bucket     string `json:"bucket"`
	ObjectName string `json:"object_name"`
}

func (c S3Server) S3() (endpoint string, secure bool, accessKey string, secretKey string, bucket string) {
	return c.Endpoint, c.Secure, c.AccessKey, c.SecretKey, c.Bucket
}

func (c S3Server) ToURL(Path string) string {
	var scheme = "https"
	if !c.Secure {
		scheme = "http"
	}
	user := url.UserPassword("s3!"+c.AccessKey, c.SecretKey)
	var u = &url.URL{
		Scheme: scheme,
		User:   user,
		Host:   c.Endpoint,
		Path:   path.Join(c.Bucket, Path),
	}
	return u.String()
}

func S3FromURL(Url string) (server *S3Server, err error) {
	var u *url.URL
	u, err = url.Parse(Url)
	if err != nil {
		return
	}
	server = new(S3Server)
	server.AccessKey = strings.TrimPrefix(u.User.Username(), "s3!")
	server.SecretKey, _ = u.User.Password()
	server.Endpoint = u.Host
	u.Path = strings.TrimPrefix(u.Path, "/")
	if u.Path == "" {
		return nil, errors.New("wrong url - no path")
	}
	paths := strings.Split(u.Path, "/")
	server.Bucket = paths[0]
	if len(paths) > 1 {
		server.ObjectName = strings.Join(paths[1:], "/")
	}
	server.Secure = u.Scheme == "https"
	return
}

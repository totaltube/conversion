package queries

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/totaltube/conversion/types"
)

func getClient(server types.S3ServerInterface) (client *minio.Client, err error) {
	endpoint, secure, accessKey, secretKey, _ := server.S3()
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 15 * time.Second,
			}).DialContext,
		},
	})
}

func StorageDelete(ctx context.Context, server types.S3ServerInterface, path string) (err error) {
	c, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()
	// Initialize minio client object.
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	objectsCh := minioClient.ListObjects(c, bucket, minio.ListObjectsOptions{Prefix: path, Recursive: true})
	errorChan := minioClient.RemoveObjects(c, bucket, objectsCh, minio.RemoveObjectsOptions{})
	for e := range errorChan {
		err = e.Err
		log.Println(e.ObjectName, err)
		return
	}
	return
}

func StorageList(ctx context.Context, server types.S3ServerInterface, path string) (list []string, err error) {
	path = strings.TrimPrefix(path, "/")
	c, cancel := context.WithTimeout(ctx, time.Minute*10)
	defer cancel()
	// Initialize minio client object.
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	list = make([]string, 0, 10)
	_, _, _, _, bucket := server.S3()
	for object := range minioClient.ListObjects(c, bucket, minio.ListObjectsOptions{Prefix: path, Recursive: true}) {
		if object.Err != nil {
			err = object.Err
			log.Println(err)
			return
		}
		list = append(list, object.Key)
	}
	return
}

func StorageGet(ctx context.Context, server types.S3ServerInterface, path string) (u *url.URL, err error) {
	// Initialize minio client object.
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	u, err = minioClient.PresignedGetObject(ctx, bucket, path, time.Hour, url.Values{})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func StorageGetObject(ctx context.Context, server types.S3ServerInterface, path string) (object *minio.Object, err error) {
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	object, err = minioClient.GetObject(ctx, bucket, path, minio.GetObjectOptions{})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func StorageFileGet(ctx context.Context, server types.S3ServerInterface, path string, savePath string) (err error) {
	c, cancel := context.WithTimeout(ctx, time.Hour*2)
	defer cancel()
	// Initialize minio client object.
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	err = minioClient.FGetObject(c, bucket, path, savePath, minio.GetObjectOptions{})
	if err != nil {
		log.Println(err, " ", bucket, " ", path, " ", savePath)
	}
	return
}

func StorageUpload(ctx context.Context, server types.S3ServerInterface, path string, body io.Reader) (err error) {
	c, cancel := context.WithTimeout(ctx, time.Hour*2)
	defer cancel()
	// Initialize minio client object.
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	var exists bool
	if exists, err = minioClient.BucketExists(ctx, bucket); err != nil {
		log.Println(err)
		return
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			log.Println(err)
			return
		}
	}
	_, err = minioClient.PutObject(c, bucket, path, body, -1, minio.PutObjectOptions{})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func StorageFileUpload(ctx context.Context, server types.S3ServerInterface, fileToUploadPath string, storagePath string) (err error) {
	c, cancel := context.WithTimeout(ctx, time.Hour*2)
	defer cancel()
	// Initialize minio client object.
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	var exists bool
	if exists, err = minioClient.BucketExists(ctx, bucket); err != nil {
		log.Println(err)
		return
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			log.Println(err)
			return
		}
	}
	_, err = minioClient.FPutObject(c, bucket, storagePath, fileToUploadPath, minio.PutObjectOptions{})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func StorageCopy(ctx context.Context, server types.S3ServerInterface, sourcePath string, destinationPath string) (err error) {
	sourcePath = strings.TrimSuffix(strings.TrimPrefix(sourcePath, "/"), "/")
	destinationPath = strings.TrimSuffix(strings.TrimPrefix(destinationPath, "/"), "/")
	if sourcePath == destinationPath {
		return
	}
	c, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	objectsCh := minioClient.ListObjects(c, bucket, minio.ListObjectsOptions{Prefix: sourcePath, Recursive: true})
	for objInfo := range objectsCh {
		var base = strings.TrimPrefix(objInfo.Key, sourcePath)
		var dest = destinationPath + base
		_, err = minioClient.CopyObject(c, minio.CopyDestOptions{
			Bucket: bucket,
			Object: dest,
		}, minio.CopySrcOptions{
			Bucket: bucket,
			Object: objInfo.Key,
		})
		if err != nil {
			log.Println(err)
			return
		}
	}
	return
}

func StorageMove(ctx context.Context, server types.S3ServerInterface, sourcePath string, destinationPath string) (err error) {
	sourcePath = strings.TrimSuffix(strings.TrimPrefix(sourcePath, "/"), "/")
	destinationPath = strings.TrimSuffix(strings.TrimPrefix(destinationPath, "/"), "/")
	if sourcePath == destinationPath {
		return
	}
	c, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	var minioClient *minio.Client
	if minioClient, err = getClient(server); err != nil {
		log.Println(err)
		return
	}
	_, _, _, _, bucket := server.S3()
	objectsCh := minioClient.ListObjects(c, bucket, minio.ListObjectsOptions{Prefix: sourcePath, Recursive: true})
	for objInfo := range objectsCh {
		var base = strings.TrimPrefix(objInfo.Key, sourcePath)
		var dest = destinationPath + base
		_, err = minioClient.CopyObject(c, minio.CopyDestOptions{
			Bucket: bucket,
			Object: dest,
		}, minio.CopySrcOptions{
			Bucket: bucket,
			Object: objInfo.Key,
		})
		if err != nil {
			log.Println(err)
			return
		}
	}
	objectsCh = minioClient.ListObjects(c, bucket, minio.ListObjectsOptions{Prefix: sourcePath, Recursive: true})
	for e := range minioClient.RemoveObjects(c, bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		err = e.Err
		log.Println(e.ObjectName, err)
		return
	}
	return
}

package helpers

import (
	"crypto/md5"
	"encoding/hex"
)


func Md5Hash(s string) (hash string) {
	h := md5.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}

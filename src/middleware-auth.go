package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func authorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorized := false
		for {
			authToken := c.GetHeader("Authorization")
			if authToken != "" {
				authToken = strings.TrimPrefix(authToken, "Bearer ")
				if authToken == conversionApiKey {
					authorized = true
					break
				}
			}
			// old auth method
			authSecret := c.GetHeader("AccessToken")
			if authSecret == "" {
				break
			}
			stringTime := c.GetHeader("Timestamp")
			if stringTime == "" {
				break
			}
			unixTime, _ := strconv.ParseInt(stringTime, 10, 64)
			if math.Abs(float64(unixTime-time.Now().Unix())) > 600 {
				// Expired
				break
			}
			authPath := c.Request.URL.Path
			if c.Request.URL.RawQuery != "" {
				authPath += "?" + c.Request.URL.RawQuery
			}
			hasher := sha1.New()
			hasher.Write([]byte(fmt.Sprintf("%d.%s.%s%s", unixTime, authPath, c.Request.Method, conversionApiKey)))
			sha := hex.EncodeToString(hasher.Sum(nil))

			if sha == authSecret {
				authorized = true
				break
			}
			// trying to decode path, maybe its urlencoded
			authPath, err := url.QueryUnescape(authPath)
			if err == nil {
				hasher := sha1.New()
				hasher.Write([]byte(fmt.Sprintf("%d.%s.%s%s", unixTime, authPath, c.Request.Method, conversionApiKey)))
				sha := hex.EncodeToString(hasher.Sum(nil))

				if sha == authSecret {
					authorized = true
				}
			}
			break
		}
		if !authorized {
			c.JSON(403, M{"success": false, "value": "not authorized"})
			c.Abort()
			return
		}
	}
}


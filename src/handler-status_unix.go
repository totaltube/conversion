//+build darwin freebsd linux

package main

import (
	"fmt"
	"log"
	"syscall"

	"github.com/davecheney/loadavg"
	"github.com/gin-gonic/gin"
)

func statusHandler(c *gin.Context) {
	var avg []string
	load, err := loadavg.LoadAvg()
	if err != nil {
		avg = []string{err.Error()}
	}
	avg = []string{
		fmt.Sprintf("%2.2f", load[loadavg.ONE_MIN]),
		fmt.Sprintf("%2.2f", load[loadavg.FIVE_MIN]),
		fmt.Sprintf("%2.2f", load[loadavg.FIFTEEN_MIN]),
	}
	var space = ""
	var stat syscall.Statfs_t

	err = syscall.Statfs(conversionPath, &stat)
	if err != nil {
		log.Println(err)
		space = err.Error()
	} else {
		space = fmt.Sprintf("%d", stat.Bavail*uint64(stat.Bsize))
	}
	c.JSON(200, M{"status": "OK", "load": avg, "space": space, "success": true, "value": M{"space": space, "load": avg}})
}

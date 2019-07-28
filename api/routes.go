package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/tectiv3/edrtsp/rtsp"
	"github.com/tectiv3/edrtsp/stats"
	"gopkg.in/go-playground/validator.v8"
)

const megabyte = 1024 * 1024

type apiHandler struct {
}

var api = &apiHandler{}

type request struct {
}

type response struct {
	Total int         `json:"total"`
	Rows  interface{} `json:"rows"`
}

func init() {
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = os.Stdout
}

//GetRouter gets gin engine
func GetRouter() *gin.Engine {
	router := gin.New()
	pprof.Register(router)
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(errors())
	router.Use(cors.Default())

	router.GET("/api/v1/serverinfo", api.GetServerInfo)
	router.POST("/api/v1/restart", api.Restart)

	router.GET("/api/v1/pushers", api.Pushers)
	router.GET("/api/v1/players", api.Players)
	return router
}

func errors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		for _, err := range c.Errors {
			switch err.Type {
			case gin.ErrorTypeBind:
				switch err.Err.(type) {
				case validator.ValidationErrors:
					errs := err.Err.(validator.ValidationErrors)
					for _, err := range errs {
						c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Sprintf("%s %s", err.Field, err.Tag))
						return
					}
				default:
					log.Println(err.Err.Error())
					c.AbortWithStatusJSON(http.StatusBadRequest, "Internal Error")
					return
				}
			}
		}
	}
}

/**
 * @api {get} /api/v1/serverinfo
 */
func (h *apiHandler) GetServerInfo(c *gin.Context) {

	mem, _ := mem.VirtualMemory()
	cpus, _ := cpu.Counts(false)

	memData, cpuData, pusherData, playerData, startTime, uptime := stats.GetStatsObject()

	c.IndentedJSON(http.StatusOK, gin.H{
		"Hardware":    strings.ToUpper(runtime.GOARCH),
		"RunningTime": uptime,
		"StartUpTime": startTime,
		"totalMemory": mem.Total / megabyte,
		"cpuCount":    fmt.Sprintf("%d", cpus),
		"Server":      fmt.Sprintf("%s for %s", "edrtsp", strings.Title(runtime.GOOS)),
		"memData":     memData,
		"cpuData":     cpuData,
		"pusherData":  pusherData,
		"playerData":  playerData,
	})
}

/**
 * @api {get} /api/v1/restart
 */
func (h *apiHandler) Restart(c *gin.Context) {
	log.Println("Restart...")
	c.JSON(http.StatusOK, "OK")
	rtsp.GetServer().Stop()
	rtsp.GetServer().Start()
}

/**
 * @api {get} /api/v1/pushers
 */
func (h *apiHandler) Pushers(c *gin.Context) {
	pushers := stats.GetPushers()
	res := response{
		Total: len(pushers),
		Rows:  pushers,
	}
	c.IndentedJSON(200, res)
}

/**
 * @api {get} /api/v1/players
 */
func (h *apiHandler) Players(c *gin.Context) {
	players := stats.GetPlayers()
	res := response{
		Total: len(players),
		Rows:  players,
	}
	c.IndentedJSON(200, res)
}

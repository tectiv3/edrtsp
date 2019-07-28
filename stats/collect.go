package stats

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/tectiv3/edrtsp/rtsp"
	"github.com/tectiv3/edrtsp/utils"
)

const megabyte = 1024 * 1024

var startTime = time.Now()
var mutex = sync.Mutex{}

type percentData struct {
	Time int64   `json:"time"`
	Used float64 `json:"used"`
}

type countData struct {
	Time  int64 `json:"time"`
	Total uint  `json:"total"`
}

var (
	memData    = make([]countData, 0)
	cpuData    = make([]percentData, 0)
	pusherData = make([]countData, 0)
	playerData = make([]countData, 0)
)

func init() {
	go collectStats()
}

func collectStats() {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("collectStats panic:%v", p)
		}
	}()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	seriesLimit := 30
	for {
		select {
		case <-ticker.C:
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)
			cpuUsage, err := cpu.Percent(0, false)
			if err != nil {
				log.Println(err)
			}
			mutex.Lock()
			now := time.Now().Unix()
			// log.Printf("mem used: %v MB, mem acquired: %v MB\n", m.Alloc/megabyte, m.Sys/megabyte)
			memData = append(memData, countData{Time: now, Total: uint(m.Sys / megabyte)})
			if len(cpuUsage) > 0 {
				cpuData = append(cpuData, percentData{Time: now, Used: cpuUsage[0] / 100})
			}
			pusherData = append(pusherData, countData{Time: now, Total: uint(rtsp.GetServer().GetPusherSize())})
			playerCnt := 0
			for _, pusher := range rtsp.GetServer().GetPushers() {
				playerCnt += len(pusher.GetPlayers())
			}
			playerData = append(playerData, countData{Time: now, Total: uint(playerCnt)})

			if len(memData) > seriesLimit {
				memData = memData[len(memData)-seriesLimit:]
			}
			if len(cpuData) > seriesLimit {
				cpuData = cpuData[len(cpuData)-seriesLimit:]
			}
			if len(pusherData) > seriesLimit {
				pusherData = pusherData[len(pusherData)-seriesLimit:]
			}
			if len(playerData) > seriesLimit {
				playerData = playerData[len(playerData)-seriesLimit:]
			}
			mutex.Unlock()
		}
	}
}

//GetStatsObject will return rtsp server and app statistics
func GetStatsObject() (interface{}, interface{}, interface{}, interface{}, time.Time, string) {
	mutex.Lock()
	defer mutex.Unlock()
	return memData, cpuData, pusherData, playerData, startTime, upTimeString()
}

//GetStats will return rtsp server and app statistics in json
func GetStats() []byte {
	mutex.Lock()
	defer mutex.Unlock()

	data := map[string]interface{}{
		"uptime":  upTimeString(),
		"mem":     memData,
		"cpu":     cpuData,
		"pushers": pusherData,
		"players": playerData,
	}

	result, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}
	return result
}

//GetPushers returns array with pushers info
func GetPushers() []interface{} {
	pushers := make([]interface{}, 0)
	ip := utils.LocalIP()
	for _, pusher := range rtsp.GetServer().GetPushers() {
		port := pusher.Server().TCPPort
		rtsp := fmt.Sprintf("rtsp://%s:%d%s", ip, port, pusher.Path())
		if port == 554 {
			rtsp = fmt.Sprintf("rtsp://%s%s", ip, pusher.Path())
		}
		pushers = append(pushers, map[string]interface{}{
			"id":        pusher.ID(),
			"url":       rtsp,
			"path":      pusher.Path(),
			"source":    pusher.Source(),
			"transType": pusher.TransType(),
			"inBytes":   pusher.InBytes(),
			"outBytes":  pusher.OutBytes(),
			"startAt":   pusher.StartAt(),
			"online":    len(pusher.GetPlayers()),
		})
	}
	return pushers
}

//GetPushersJSON returns json encoded pushers
func GetPushersJSON() []byte {
	pushers := GetPushers()

	result, err := json.Marshal(pushers)
	if err != nil {
		log.Println(err)
	}
	return result
}

//GetPlayers returns array with players info
func GetPlayers() []interface{} {
	players := make([]*rtsp.Player, 0)
	for _, pusher := range rtsp.GetServer().GetPushers() {
		for _, player := range pusher.GetPlayers() {
			players = append(players, player)
		}
	}
	ip := utils.LocalIP()
	_players := make([]interface{}, 0)
	for i := 0; i < len(players); i++ {
		player := players[i]
		port := player.Server.TCPPort
		rtsp := fmt.Sprintf("rtsp://%s:%d%s", ip, port, player.Path)
		if port == 554 {
			rtsp = fmt.Sprintf("rtsp://%s%s", ip, player.Path)
		}
		_players = append(_players, map[string]interface{}{
			"id":        player.ID,
			"path":      rtsp,
			"transType": player.TransType.String(),
			"inBytes":   player.InBytes,
			"outBytes":  player.OutBytes,
			"startAt":   player.StartAt,
		})
	}
	return _players
}

//GetPlayersJSON returns json encoded players
func GetPlayersJSON() []byte {
	players := GetPlayers()

	result, err := json.Marshal(players)
	if err != nil {
		log.Println(err)
	}
	return result
}

func upTime() time.Duration {
	return time.Since(startTime)
}

func upTimeString() string {
	d := upTime()
	days := d / (time.Hour * 24)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second
	return fmt.Sprintf("%d Days %d Hours %d Mins %d Secs", days, hours, minutes, seconds)
}

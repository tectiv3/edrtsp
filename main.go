package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tectiv3/edrtsp/api"
	"github.com/tectiv3/edrtsp/rtsp"
	"github.com/tectiv3/edrtsp/utils"
)

var (
	// GitCommitCode auto set on build
	GitCommitCode string
	// BuildDateTime auto set on build
	BuildDateTime string
	// BuildVersion version
	BuildVersion = "v0.1.1"
)

type program struct {
	httpPort   int
	httpServer *http.Server
	rtspPort   int
	rtspServer *rtsp.Server
}

func (p *program) stopHTTP() (err error) {
	if p.httpServer == nil {
		err = fmt.Errorf("HTTP Server Not Found")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = p.httpServer.Shutdown(ctx); err != nil {
		return
	}
	return
}

func (p *program) startHTTP() (err error) {
	p.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", p.httpPort),
		Handler:           api.GetRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	link := fmt.Sprintf("http://%s:%d", utils.LocalIP(), p.httpPort)
	log.Println("http server started -->", link)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("http panic ocurs:%v", p)
			}
		}()
		if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("http server start failed", err)
		}
		log.Println("http server stopped")
	}()
	return
}

func (p *program) startRTSP() (err error) {
	if p.rtspServer == nil {
		err = fmt.Errorf("RTSP Server Not Found")
		return
	}
	sport := ":554"
	if p.rtspPort != 554 {
		sport = fmt.Sprintf(":%d", p.rtspPort)
	}
	link := fmt.Sprintf("rtsp://%s%s", utils.LocalIP(), sport)
	log.Println("rtsp server started -->", link)
	go func() {
		if err := p.rtspServer.Start(); err != nil {
			log.Println("rtsp server start failed", err)
		}
		log.Println("rtsp server stopped")
	}()
	return
}

func (p *program) stopRTSP() (err error) {
	if p.rtspServer == nil {
		err = fmt.Errorf("RTSP Server Not Found")
		return
	}
	p.rtspServer.Stop()
	return
}

type stream struct {
	URL               string `gorm:"type:varchar(256);primary_key;unique"`
	CustomPath        string `gorm:"type:varchar(256)"`
	IdleTimeout       int
	HeartbeatInterval int
}

func (p *program) start() (err error) {
	log.Println("********** START **********")
	if utils.IsPortInUse(p.rtspPort) {
		err = fmt.Errorf("RTSP port[%d] In Use", p.rtspPort)
		return
	}

	p.startRTSP()
	p.startHTTP()

	log.SetOutput(os.Stdout)

	go func() {
		streams := []stream{}
		log.Printf("demon pull streams %d\n", len(streams))
		for {
			// streams = append(streams, stream{
			// 	"rtsp://localhost:8554/roi",
			// 	"roi",
			// 	1,
			// 	10,
			// })
			for i := len(streams) - 1; i > -1; i-- {
				v := streams[i]
				agent := fmt.Sprintf("edrtsp/%s", "0.0.1")
				if BuildDateTime != "" {
					agent = fmt.Sprintf("%s(%s)", agent, BuildDateTime)
				}
				client, err := rtsp.NewRTSPClient(rtsp.GetServer(), v.URL, int64(v.HeartbeatInterval)*1000, agent)
				if err != nil {
					continue
				}
				client.CustomPath = v.CustomPath

				pusher := rtsp.NewClientPusher(client)
				if rtsp.GetServer().GetPusher(pusher.Path()) != nil {
					continue
				}
				err = client.Start(time.Duration(v.IdleTimeout) * time.Second)
				if err != nil {
					log.Printf("Pull stream err :%v", err)
					continue
				}
				rtsp.GetServer().AddPusher(pusher)
				//streams = streams[0:i]
				//streams = append(streams[:i], streams[i+1:]...)
			}
			time.Sleep(10 * time.Second)
		}
	}()
	return
}

func (p *program) stop() (err error) {
	defer log.Println("********** STOP **********")
	p.stopHTTP()
	p.stopRTSP()
	return
}

func main() {
	// log
	log.SetPrefix("[edrtsp] ")
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	log.Printf("git commit code:%s", GitCommitCode)
	log.Printf("build date:%s", BuildDateTime)

	rtspServer := rtsp.GetServer()
	p := &program{
		rtspPort:   rtspServer.TCPPort,
		rtspServer: rtspServer,
		httpPort:   8080,
	}
	p.start()

	select {}
}

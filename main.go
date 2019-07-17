package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/tectiv3/edrtsp/rtsp"
)

var (
	gitCommitCode string
	buildDateTime string
)

type program struct {
	rtspPort   int
	rtspServer *rtsp.Server
}

func localIP() string {
	ip := ""
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsMulticast() && !ipnet.IP.IsLinkLocalUnicast() && !ipnet.IP.IsLinkLocalMulticast() && ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}
	return ip
}

func isPortInUse(port int) bool {
	if conn, err := net.DialTimeout("tcp", net.JoinHostPort("", fmt.Sprintf("%d", port)), 3*time.Second); err == nil {
		conn.Close()
		return true
	}
	return false
}

func (p *program) StartRTSP() (err error) {
	if p.rtspServer == nil {
		err = fmt.Errorf("RTSP Server Not Found")
		return
	}
	sport := ":554"
	if p.rtspPort != 554 {
		sport = fmt.Sprintf(":%d", p.rtspPort)
	}
	link := fmt.Sprintf("rtsp://%s%s", localIP(), sport)
	log.Println("rtsp server start -->", link)
	go func() {
		if err := p.rtspServer.Start(); err != nil {
			log.Println("start rtsp server error", err)
		}
		log.Println("rtsp server stopped")
	}()
	return
}

func (p *program) StopRTSP() (err error) {
	if p.rtspServer == nil {
		err = fmt.Errorf("RTSP Server Not Found")
		return
	}
	p.rtspServer.Stop()
	return
}

func (p *program) Start() (err error) {
	log.Println("********** START **********")
	if isPortInUse(p.rtspPort) {
		err = fmt.Errorf("RTSP port[%d] In Use", p.rtspPort)
		return
	}

	p.StartRTSP()

	log.SetOutput(os.Stdout)

	// go func() {
	//     log.Printf("demon pull streams")
	//     for {
	//         var streams []string
	//
	//         for i := len(streams) - 1; i > -1; i-- {
	//             v := streams[i]
	//             agent := fmt.Sprintf("EasyDarwinGo/%s", routers.BuildVersion)
	//             if routers.BuildDateTime != "" {
	//                 agent = fmt.Sprintf("%s(%s)", agent, routers.BuildDateTime)
	//             }
	//             client, err := rtsp.NewRTSPClient(rtsp.GetServer(), v.URL, int64(v.HeartbeatInterval)*1000, agent)
	//             if err != nil {
	//                 continue
	//             }
	//             client.CustomPath = v.CustomPath
	//
	//             pusher := rtsp.NewClientPusher(client)
	//             if rtsp.GetServer().GetPusher(pusher.Path()) != nil {
	//                 continue
	//             }
	//             err = client.Start(time.Duration(v.IdleTimeout) * time.Second)
	//             if err != nil {
	//                 log.Printf("Pull stream err :%v", err)
	//                 continue
	//             }
	//             rtsp.GetServer().AddPusher(pusher)
	//             //streams = streams[0:i]
	//             //streams = append(streams[:i], streams[i+1:]...)
	//         }
	//         time.Sleep(10 * time.Second)
	//     }
	// }()
	return
}

func (p *program) Stop() (err error) {
	defer log.Println("********** STOP **********")
	p.StopRTSP()
	return
}

func main() {
	// log
	log.SetPrefix("[edrtsp] ")
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	log.Printf("git commit code:%s", gitCommitCode)
	log.Printf("build date:%s", buildDateTime)

	rtspServer := rtsp.GetServer()
	p := &program{
		rtspPort:   rtspServer.TCPPort,
		rtspServer: rtspServer,
	}
	p.Start()

	select {}
}

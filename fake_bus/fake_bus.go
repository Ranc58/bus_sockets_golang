package main

import (
	"bus_sockets/buses"
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"
)

type BusInfo struct {
	Ctx  context.Context
	Info *buses.RouteInfo
}

type BusImitator struct {
	ctx context.Context
	serverAddress string
	refreshTimeout int
	routesCount int
	busInfoChans []chan *buses.BusRouteData
	busesPerRoute int
	wg *sync.WaitGroup
}


func (b *BusImitator) initWs(readyWs chan struct{}){
	wg := sync.WaitGroup{}
	for i := 0; i < *wsCount; i++ {
		busInfoCh := make(chan *buses.BusRouteData, 0)
		b.busInfoChans = append(b.busInfoChans, busInfoCh)
		wg.Add(1)
		go b.spawnBusFromCh(busInfoCh, &wg)
	}
	readyWs <- struct {}{}
	wg.Wait()
}


func (b *BusImitator) processBuses(files []os.FileInfo, routesDir string) {
	wg := sync.WaitGroup{}
	for i := 0; i < b.routesCount; i++ {
		fileInfo := files[i]
		rand.Seed(time.Now().UnixNano())
		busInfoChan := b.busInfoChans[rand.Intn(len(b.busInfoChans))]
		wg.Add(1)
		go b.spawnRoute(routesDir, fileInfo, busInfoChan, &wg)
	}
	wg.Wait()
}

func (b *BusImitator) spawnBusFromCh(
	busInfoCh <-chan *buses.BusRouteData,
	wg *sync.WaitGroup,
) {
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:8080", Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	done := make(chan struct{})
	ticker := time.NewTicker(time.Duration(b.refreshTimeout) * time.Millisecond)
	defer func() {
		wg.Done()
		c.Close()
		close(done)
		ticker.Stop()
	}()
	go func() {

		for {
			select {
			case <-b.ctx.Done():
				return
			default:
				_, _, err := c.ReadMessage()
				if err != nil {
					return
				}
			}

		}
	}()
	for {
		select {
		case busInfo := <-busInfoCh:
			msg, err := busInfo.MarshalJSON()
			if err != nil{
				log.Println("marshal error:", err)
				return
			}
			c.SetWriteDeadline(time.Now().Add(time.Second * 2))
			err = c.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				log.Println("write send:", err)
				return
			}
			<-ticker.C
		case <-done:
			return
		case <-b.ctx.Done():
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}

}

func (b *BusImitator) sendBusToCh(
	InfoSender *BusInfo,
	busInfoCh chan<- *buses.BusRouteData,
	sendBusWg *sync.WaitGroup,
) {
	defer func() {
		sendBusWg.Done()
	}()
	busId := fmt.Sprintf("%s-%s", InfoSender.Info.Name, uuid.New().String()[:5])
	firstRun := true
	coords := InfoSender.Info.Coordinates
	for {

		if firstRun {
			randOffset := rand.Intn(len(coords) / 2)
			coords = coords[randOffset:]
		}
		fmt.Printf("%s-%t %d\n", busId, firstRun, len(coords))
		for _, coord := range coords {
			busData := buses.BusRouteData{
				BusID: busId,
				Lat:   coord[0],
				Lng:   coord[1],
				Route: InfoSender.Info.Name,
			}
			select {
			case <-b.ctx.Done():
				return
			case busInfoCh <- &busData:
			}
		}

		if firstRun{
			firstRun = false
			coords = InfoSender.Info.Coordinates
		}
		for i := len(coords)/2 - 1; i >= 0; i-- {
			opp := len(coords) - 1 - i
			coords[i], coords[opp] = coords[opp], coords[i]
		}
	}
}

func (b *BusImitator) spawnRoute(
	routesDir string,
	fileInfo os.FileInfo,
	busInfoCh chan<- *buses.BusRouteData,
	wg *sync.WaitGroup,
) {
	sendBusWg := sync.WaitGroup{}
	defer func() {
		sendBusWg.Wait()
		wg.Done()
	}()
	fullPath := path.Join(routesDir, fileInfo.Name())
	f, err := os.Open(fullPath)
	if err != nil {
		log.Printf("unable to open file: %s\n", err)
		return
	}
	fileContent, err := ioutil.ReadAll(f)
	if err != nil {
		log.Printf("unable to open file: %s\n", err)
		return
	}
	_ = f.Close()
	data := buses.RouteInfo{}
	err = data.UnmarshalJSON(fileContent)
	if err != nil {
		log.Printf("unable to unmarshal json: %s\n", err)
		return
	}
	InfoSender := BusInfo{
		Info: &data,
	}
	for i := 0; i < b.busesPerRoute; i++ {
		sendBusWg.Add(1)
		go b.sendBusToCh(&InfoSender, busInfoCh, &sendBusWg)
	}
}

var serverAddr = flag.String("server", "127.0.0.1:8080", "Address:port of gate service")
var routesCount = flag.Int("routes", 10, "Count of routes")
var busesPerRoute = flag.Int("buses", 10, "Count of buses on one route")
var wsCount = flag.Int("sockets", 10, "Count of WebSockets")
var refreshTimeout = flag.Int("refresh", 500, "Refresh timeout (on milliseconds)")

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	shutDownCh := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(shutDownCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-shutDownCh
		log.Printf("Shutdown by signal: %s", sig)
		cancel()
		time.Sleep(1 * time.Second)
		done <- true
	}()

	dir, err := os.Getwd()
	if err != nil {
		log.Printf("unable to get current directory: %s\n", err)
		return
	}

	routesDir := path.Join(dir, "routes")
	files, err := ioutil.ReadDir(routesDir)

	if err != nil {
		log.Printf("unable to read routes directory: %s\n", err)
		return
	}

	imitator := BusImitator{
		ctx:            ctx,
		serverAddress:  *serverAddr,
		refreshTimeout: *refreshTimeout,
		routesCount: *routesCount,
		busesPerRoute:  *busesPerRoute,
		busInfoChans: []chan *buses.BusRouteData{},
	}
	readyWs := make(chan struct{})
	go imitator.initWs(readyWs)
	<-readyWs
	fmt.Printf("Start listen server as: 127.0.0.1:8080\n")
	imitator.processBuses(files, routesDir)
	<-done
	fmt.Println("DONE OK")
}

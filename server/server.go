package main

import (
	"bus_sockets/buses"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var upgrader = websocket.Upgrader{}
//var busesInfoToSend = map[string]buses.BusRouteData{}
var busesInfoToSend = sync.Map{}


type MainHandler struct {
	Ctx context.Context
	mx sync.RWMutex
}

type ListenHandler struct {
	Ctx context.Context
	mx sync.Mutex
}


func sendBusInfo(
	ctx context.Context,
	ws *websocket.Conn,
	busesData *buses.BusesData,
	) error {
	select {
	case <- ctx.Done():
		ws.SetWriteDeadline(time.Now().Add(time.Second * 2))
		err := ws.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			return err
		}
		select {
		case <-time.After(time.Second):
		}
		return nil
	default:
		var busesRouteData []buses.BusRouteData
		busesInfoToSend.Range(func(key, value interface{}) bool {
			val, ok := value.(buses.BusRouteData)
			if !ok {
				return false
			}
			busesRouteData = append(busesRouteData, val)
			return true

		})
		busesData.Buses = busesRouteData
		ws.SetWriteDeadline(time.Now().Add(time.Second * 2))
		err := ws.WriteJSON(&busesData)
		if err != nil {
			return err
		}
		return nil
	}
}

func (m *MainHandler) wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer ws.Close()
	busesData := buses.BusesData{
		MsgType: "Buses",
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := sendBusInfo(m.Ctx, ws, &busesData)
		if err != nil {
			return
		}
		<-ticker.C
		}
	}


func getBusInfo(ctx context.Context, ws *websocket.Conn) error{
	
	select {
	case <- ctx.Done():
		return nil
	default:
		busData := buses.BusRouteData{}
		err := ws.ReadJSON(&busData)
		if err != nil {
			return err
		}
		fmt.Println(busData)
		busesInfoToSend.Store(busData.BusID, busData)
	}
	return nil
}

func (l *ListenHandler) listenHandler(w http.ResponseWriter, r *http.Request){
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer ws.Close()
	for {
		err := getBusInfo(l.Ctx, ws)
		if err != nil {
			return
		}
	}
}


func main() {

	ctx, cancel := context.WithCancel(context.Background())
	shutDownCh := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(shutDownCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-shutDownCh
		log.Printf("Shutdown by signal: %s", sig)
		cancel()
		time.Sleep(2 * time.Second)
		done <- true
	}()


	go func() {

		mHandler := MainHandler{Ctx:ctx}
		http.HandleFunc("/ws", mHandler.wsHandler)
		log.Println("Start ws server as :8000")
		http.ListenAndServe(":8000", nil)
	}()
	go func() {

		lHandler := ListenHandler{Ctx:ctx}
		http.HandleFunc("/", lHandler.listenHandler)
		log.Println("Start listen server as :8080")
		http.ListenAndServe(":8080", nil)
	}()

	<- done
}

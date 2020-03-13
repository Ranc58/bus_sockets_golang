# Bus routes
Map with Moscow bus routes. 
Writing only for Golang training and fun. Based on websockets and uses 
server for sending data to browser and gateway imitator for load bus routes, spawn some buses and send then to server.\
![](pics/buses.gif)
# How to run
For run server `go run server/server`
For run bus imitator `go run fake_bus/bus.go`. With imitator you can use some flags:
```
  -buses int
        Count of buses on one route (default 10)
  -refresh int
        Refresh timeout (on milliseconds) (default 500)
  -routes int
        Count of routes (default 10)
  -server string
        Address:port of gate service (default "127.0.0.1:8080")
  -sockets int
        Count of WebSockets (default 10)

```

And open `frontend/index.html` on your browser.
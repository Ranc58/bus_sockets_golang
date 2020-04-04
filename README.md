# Bus routes
Map with Moscow bus routes. 
Writing only for Golang training and fun. Based on websockets and RabbitMQ and uses 
server for sending data to browser and gateway imitator for load bus routes, spawn some buses and send then to server.\
![](pics/buses.gif)
# How to run
First you need RabbitMQ. You can run it from docker  `docker run -p 127.0.0.1:5672:5672/tcp  -e RABBITMQ_DEFAULT_USER="rabbitmq" -e RABBITMQ_DEFAULT_PASS="rabbitmq" rabbitmq:latest`
For run server `go run server/server`
For run bus imitator `go run fake_bus/bus.go`. With imitator you can use some flags:
```
    -buses int
        Count of buses on one route (default 5)
    -chans int
        Count of parallel Golang chans for send bus data to rabbit (default 10)
    -r_host string
        RabbitMQ host (default "127.0.0.1")
    -r_login string
        RabbitMQ login (default "rabbitmq")
    -r_pass string
        RabbitMQ password (default "rabbitmq")
    -r_port int
        RabbitMQ port (default 5672)
    -refresh int
        Refresh timeout (on milliseconds) (default 100)
    -routes int
        Count of routes (default 20)

```

And open `frontend/index.html` on your browser.
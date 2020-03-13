package buses

type Point [2]float64

type RouteInfo struct {
	Name             string   `json:"name"`
	StationStartName string   `json:"station_start_name"`
	StationStopName  string   `json:"station_stop_name"`
	Coordinates      []*Point `json:"coordinates"`
}

type BusesData struct {
	MsgType string         `json:"msgType"`
	Buses   []BusRouteData `json:"buses"`
}

type BusRouteData struct {
	BusID string  `json:"busId"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Route string  `json:"route"`
}

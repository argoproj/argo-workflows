// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package axops

type App struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Stage struct {
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

type PerfData struct {
	Time int64   `json:"time"`
	Data float64 `json:"data"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

type PerfDataBreakDown struct {
	Time     int64   `json:"time"`
	Data     float64 `json:"data"`
	Name     string  `json:"name"`
	Id       string  `json:"id"`
	IsSystem bool    `json:"is_system"`
}

type Usage struct {
	Desc        string                 `json:"desc"`
	Spent       float64                `json:"spent"`
	Utilization float64                `json:"utilization"`
	CostID      map[string]interface{} `json:"cost_id"`
}

//type Host struct {
//	Id       string    `json:"id"`
//	Name     string    `json:"name"`
//	Status   int       `json:"status"` // see constants host status
//	Mem      int       `json:"mem"`
//	CPU      int       `json:"cpu"`
//	CpuUsage float32   `json:"cpu_usage"`
//	Services []Service `json:"services"`
//}

type Cluster struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Status   int      `json:"status"` // see constants host status
	Hosts    int      `json:"hosts"`
	Services int      `json:"services"`
	Tags     []string `json:"tags"`
	Cost     float64  `json:"cost"`
	VPC      string   `json:"vpc"`
	Region   string   `json:"region"`
}

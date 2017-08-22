package monitor

import (
	"encoding/json"
	"log"
	"runtime"
	"time"
)

type SystemProfile struct {
	NumCgoCall   int64
	NumCPU       int
	NumGoroutine int
	Memory       *SystemMemory
}

type SystemMemory struct {
	// General statistics.
	Alloc      uint64 // bytes allocated and not yet freed
	TotalAlloc uint64 // bytes allocated (even if freed)
	Sys        uint64 // bytes obtained from system (sum of XxxSys below)
	Lookups    uint64 // number of pointer lookups
	Mallocs    uint64 // number of mallocs
	Frees      uint64 // number of frees

	// Main allocation heap statistics.
	HeapAlloc    uint64 // bytes allocated and not yet freed (same as Alloc above)
	HeapSys      uint64 // bytes obtained from system
	HeapIdle     uint64 // bytes in idle spans
	HeapInuse    uint64 // bytes in non-idle span
	HeapReleased uint64 // bytes released to the OS
	HeapObjects  uint64 // total number of allocated objects

	// Low-level fixed-size structure allocator statistics.
	//	Inuse is bytes used now.
	//	Sys is bytes obtained from system.
	StackInuse  uint64 // bytes used by stack allocator
	StackSys    uint64
	MSpanInuse  uint64 // mspan structures
	MSpanSys    uint64
	MCacheInuse uint64 // mcache structures
	MCacheSys   uint64
	BuckHashSys uint64 // profiling bucket hash table
	GCSys       uint64 // GC metadata
	OtherSys    uint64 // other system allocations

	// Garbage collector statistics.
	NextGC       uint64 // next collection will happen when HeapAlloc ≥ this amount
	LastGC       uint64 // end time of last collection (nanoseconds since 1970)
	PauseTotalNs uint64

	NumGC         uint32
	GCCPUFraction float64 // fraction of CPU time used by GC
	EnableGC      bool
	DebugGC       bool
}

func (mm *SystemMemory) fromMemStats(m *runtime.MemStats) {
	mm.Alloc = m.Alloc
	mm.TotalAlloc = m.TotalAlloc
	mm.Sys = m.Sys
	mm.Lookups = m.Lookups
	mm.Mallocs = m.Mallocs
	mm.Frees = m.Frees

	mm.HeapAlloc = m.HeapAlloc
	mm.HeapSys = m.HeapSys
	mm.HeapIdle = m.HeapIdle
	mm.HeapInuse = m.HeapInuse
	mm.HeapReleased = m.HeapReleased
	mm.HeapObjects = m.HeapObjects

	mm.StackInuse = m.StackInuse
	mm.StackSys = m.StackSys
	mm.MSpanInuse = m.MSpanInuse
	mm.MSpanSys = m.MSpanSys
	mm.MCacheInuse = m.MCacheInuse
	mm.MCacheSys = m.MCacheSys
	mm.BuckHashSys = m.BuckHashSys
	mm.GCSys = m.GCSys
	mm.OtherSys = m.OtherSys

	mm.NextGC = m.NextGC
	mm.LastGC = m.LastGC
	mm.PauseTotalNs = m.PauseTotalNs
	mm.NumGC = m.NumGC
	mm.GCCPUFraction = m.GCCPUFraction
	mm.EnableGC = m.EnableGC
	mm.DebugGC = m.DebugGC
}

func MonitorSystem(interval time.Duration, logger *log.Logger) {
	ticker := time.NewTicker(interval)
	go func() {
		for _ = range ticker.C {
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)
			systemMem := &SystemMemory{}
			systemMem.fromMemStats(m)
			profile := SystemProfile{
				NumCgoCall:   runtime.NumCgoCall(),
				NumCPU:       runtime.NumCPU(),
				NumGoroutine: runtime.NumGoroutine(),
				Memory:       systemMem,
			}
			str, err := json.Marshal(profile)
			if err == nil {
				logger.Println(string(str))
			} else {
				logger.Printf("Marshal system profile failed: %v\n", err)
			}
		}
	}()
}

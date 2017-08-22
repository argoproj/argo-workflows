package heartbeat

import "applatix.io/axerror"
import (
	"sync"

	"time"

	"applatix.io/common"
)

type HeartBeat struct {
	Date   int64                  `json:"date,omitempty"`
	Key    string                 `json:"key,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
	Origin []byte                 `json:"origin"`
}

type HeartBeatHandler func(*HeartBeat) *axerror.AXError

var handlerMap = map[string]HeartBeatHandler{}
var handlerLock sync.RWMutex

func RegisterHandler(key string, handler HeartBeatHandler) {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	handlerMap[key] = handler

	heartBeatFreshLock.Lock()
	defer heartBeatFreshLock.Unlock()
	heartBeatFreshMap[key] = time.Now().Unix()

	common.InfoLog.Printf("[HB] Register key %v.\n", key)
}

func UnregisterHandler(key string) {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	delete(handlerMap, key)

	heartBeatFreshLock.Lock()
	defer heartBeatFreshLock.Unlock()
	delete(heartBeatFreshMap, key)

	common.InfoLog.Printf("[HB] Unregister key %v.\n", key)
}

func GetHandler(key string) HeartBeatHandler {
	handlerLock.RLock()
	defer handlerLock.RUnlock()
	return handlerMap[key]
}

var heartBeatFreshMap = map[string]int64{}
var heartBeatFreshLock sync.RWMutex

func UpdateFreshness(key string, time int64) {
	heartBeatFreshLock.Lock()
	defer heartBeatFreshLock.Unlock()

	if time > heartBeatFreshMap[key] {
		heartBeatFreshMap[key] = time
	}
}

func GetFreshness(key string) int64 {
	heartBeatFreshLock.RLock()
	defer heartBeatFreshLock.RUnlock()
	return heartBeatFreshMap[key]
}

func ProcessHeartBeat(hb *HeartBeat) *axerror.AXError {
	if hb == nil {
		return nil
	}

	handler := GetHandler(hb.Key)
	if handler == nil {
		common.ErrorLog.Printf("[HB]HeartBeat Drop: Cannot find heartbeat handler for key %v.\n", hb.Key)
		common.DebugLog.Println("[HB]", string(hb.Origin))
		return nil
	}

	UpdateFreshness(hb.Key, hb.Date)

	axErr := handler(hb)
	if axErr != nil {
		return axErr
	}

	return nil
}

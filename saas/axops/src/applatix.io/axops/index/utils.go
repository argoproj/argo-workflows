package index

import (
	"strings"
	"sync"
	"time"

	"applatix.io/axops/utils"
)

var indexChan = make(chan *SearchIndex, 20000)
var indexDedupMapUpdateTime = time.Now().Unix()
var indexDedupMap = map[string]interface{}{}
var indexDedupMapLock sync.Mutex

func SearchIndexWorker() {
	for profile := range indexChan {
		utils.DebugLog.Printf("Profile buffer channel status: %v/%v\n", len(indexChan), cap(indexChan))
		_, dbErr, _ := profile.Create()
		if dbErr != nil && !strings.Contains(dbErr.Message, "Can't POST the same entry twice") {
			utils.InfoLog.Printf("[INDEX] Update search index failed: %v", dbErr)
		}
	}
}

func SendToSearchIndexChan(typeStr, key, val string) {
	indexDedupMapLock.Lock()
	defer indexDedupMapLock.Unlock()
	if _, ok := indexDedupMap[typeStr+"$$"+key+"$$"+val]; ok {
		return
	}

	p := &SearchIndex{
		Type:  typeStr,
		Key:   key,
		Value: val,
	}

	select {
	case indexChan <- p:
	default:
		utils.DebugLog.Printf("[INDEX] channel is full, data: %v\n", *p)
	}

	indexDedupMap[typeStr+"$$"+key+"$$"+val] = nil
	if len(indexDedupMap) > 4000 || time.Now().Unix() > indexDedupMapUpdateTime+86400 {
		indexDedupMap = map[string]interface{}{}
		indexDedupMapUpdateTime = time.Now().Unix()
		utils.DebugLog.Printf("[INDEX] indexDedupMap is rotated.\n")
	}
}

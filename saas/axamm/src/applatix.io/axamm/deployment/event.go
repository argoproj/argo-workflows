package deployment

import (
	"applatix.io/axamm/utils"
	"applatix.io/axerror"
	"strings"
	"sync"
	"time"
)

var statusMutex sync.Mutex

var statusIdChannels map[interface{}]*StatusIdsChannel = map[interface{}]*StatusIdsChannel{}

type StatusIdsChannel struct {
	Ch     chan *utils.Event
	Filter map[string]interface{}
}

func (c *StatusIdsChannel) Match(id string) bool {
	if c.Filter == nil {
		return true
	}

	_, ok := c.Filter[strings.ToLower(id)]
	return ok
}

func (c *StatusIdsChannel) AddFilter(ids []string) {
	if ids == nil || len(ids) == 0 {
		return
	}

	filter := map[string]interface{}{}
	for _, id := range ids {
		filter[strings.ToLower(id)] = nil
	}

	c.Filter = filter
}

func GetStatusServiceIdsChannel(ctx interface{}, ids []string) (<-chan *utils.Event, *axerror.AXError) {

	ch := make(chan *utils.Event, 10)
	channel := &StatusIdsChannel{}
	channel.Ch = ch
	channel.AddFilter(ids)

	statusMutex.Lock()
	defer statusMutex.Unlock()

	utils.DebugLog.Printf("[STREAM] GetStatusBranchChannel branches %v ctx %v channel %v", ids, ctx, ch)
	statusIdChannels[ctx] = channel

	utils.DebugLog.Printf("[STREAM] channel map size: %d", len(statusIdChannels))

	return ch, nil
}

func ClearStatusIdsChannel(ctx interface{}) *axerror.AXError {
	statusMutex.Lock()
	defer statusMutex.Unlock()

	if statusIdChannels[ctx] == nil {
		return nil
	}

	utils.DebugLog.Printf("[STREAM] ClearStatusBranchChannel ctx %v channel %v", ctx, statusIdChannels[ctx])
	delete(statusIdChannels, ctx)

	utils.DebugLog.Printf("[STREAM] channel map size: %d", len(statusIdChannels))
	return nil
}

func PostStatusEvent(id, name, status string, detail map[string]interface{}) {
	event := utils.Event{
		Id:           id,
		Status:       status,
		Name:         name,
		StatusDetail: detail,
	}

	retryCount := 0
	for {
		retry := false
		statusMutex.Lock()

		for _, ch := range statusIdChannels {
			if ch != nil && ch.Match(id) {
				if ch.Ch != nil {
					select {
					case ch.Ch <- &event:
						utils.DebugLog.Printf("[STREAM] PostServiceStatusEvent id %v name %v status %v to channel %v", id, name, status, ch)
					default:
						utils.DebugLog.Printf("[STREAM] PostServiceStatusEvent id %v name %v status %v to channel %v, operation failed", id, name, status, ch)
						retry = true
						break
					}
				}
			}
		}
		statusMutex.Unlock()

		if retry && retryCount < 300 {
			time.Sleep(100 * time.Millisecond)
			retryCount++
		} else {
			break
		}
	}

	utils.DebugLog.Printf("[STREAM] channel map size: %d", len(statusIdChannels))
}

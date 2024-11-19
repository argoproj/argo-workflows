package templateresolution

import (
	"context"

	"github.com/argoproj/pkg/sync"
	log "github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/workqueue"
)

type wftmplStatusQueue struct {
	wftmplQueue  workqueue.RateLimitingInterface
	cwftmplQueue workqueue.RateLimitingInterface

	keyLock  sync.KeyLock
	ckeyLock sync.KeyLock
	ctx      *Context
}

func NewTmplStatusQueue(ctx *Context) *wftmplStatusQueue {
	return &wftmplStatusQueue{
		wftmplQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "tmpl-status-queue"),
		cwftmplQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ctmpl-status-queue"),
		keyLock:      sync.NewKeyLock(),
		ckeyLock:     sync.NewKeyLock(),
		ctx:          ctx,
	}
}

func (q *wftmplStatusQueue) run(ctx context.Context) {
	defer q.wftmplQueue.ShutDown()
	defer q.cwftmplQueue.ShutDown()
	go q.runTmplStatusUpdate()
	go q.runCtmplStatusUpdate()
	<-ctx.Done()
}

func (q *wftmplStatusQueue) runTmplStatusUpdate() {
	ctx := context.TODO()
	for q.processNextTmplItem(ctx) {
	}
}

func (q *wftmplStatusQueue) runCtmplStatusUpdate() {
	ctx := context.TODO()
	for q.processNextCtmplItem(ctx) {
	}
}

func (q *wftmplStatusQueue) processNextTmplItem(ctx context.Context) bool {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	key, quit := q.wftmplQueue.Get()
	if quit {
		return false
	}
	defer q.wftmplQueue.Done(key)

	q.keyLock.Lock(key.(string))
	defer q.keyLock.Unlock(key.(string))

	logCtx := log.WithField("wftmplStatus", key)
	logCtx.Infof("Processing %s", key)

	err := q.ctx.updateTemplateStatus(ctx, key.(string))
	if err != nil {
		log.Errorf("Update workflow template %s err: %v", key.(string), err)
	}
	return true
}

func (q *wftmplStatusQueue) processNextCtmplItem(ctx context.Context) bool {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	key, quit := q.cwftmplQueue.Get()
	if quit {
		return false
	}
	defer q.cwftmplQueue.Done(key)

	q.ckeyLock.Lock(key.(string))
	defer q.ckeyLock.Unlock(key.(string))

	logCtx := log.WithField("cwftmplStatus", key)
	logCtx.Infof("Processing %s", key)

	err := q.ctx.updateCtemplateStatus(ctx, key.(string))
	if err != nil {
		log.Errorf("Update cluster workflow template %s err: %v", key.(string), err)
	}
	return true
}

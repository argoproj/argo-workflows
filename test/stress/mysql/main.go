package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/argoproj/pkg/rand"
	"github.com/hashicorp/go-uuid"
	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/mysql"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
)

type archivedRecord struct {
	StartedAt time.Time `db:"startedat"`
}

func getFirstArchivedRecord(session db.Session) (*archivedRecord, error) {
	var record *archivedRecord
	err := session.SQL().Select("startedat").
		From("argo_archived_workflows").
		OrderBy("startedat asc").
		Limit(1).
		One(&record)
	if err != nil {
		log.Warnf("Get first archived workflow error: %s", err)
	}
	return record, err
}

func getLastArchivedRecord(session db.Session) (*archivedRecord, error) {
	var record *archivedRecord
	err := session.SQL().Select("startedat").
		From("argo_archived_workflows").
		OrderBy("startedat desc").
		Limit(1).
		One(&record)
	if err != nil {
		log.Warnf("Get last archived workflow error: %s", err)
	}
	return record, err
}

func getArchivedWorkflowsCount(session db.Session) (int, error) {
	var count *int
	rows, err := session.SQL().Query("select count(*) count from argo_archived_workflows")
	if err != nil {
		log.Warnf("Get archived workflow count error: %s", err)
		return 0, err
	}
	defer func() {
		rows.Close()
	}()
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			log.Warnf("Scan archived workflow count error: %s", err)
			return 0, err
		}
		return *count, nil
	}
	return 0, rows.Err()
}

// Related environment variables:
// - WORKFLOW_GC_PERIOD
// - OFFLOAD_NODE_STATUS_TTL
func main() {
	var dbUri string
	var parallel int
	var duration time.Duration
	var archiveCount int
	var archiveCleanSize int
	var nodeStatusSize int
	var archiveNodeStatusSize int
	var archiveWorkers int

	flag.StringVar(&dbUri, "db-uri", "root:root@tcp(localhost:3306)/argoperftest", "Database connection url string")
	flag.IntVar(&parallel, "parallel", 100, "Concurrent worker count")
	flag.DurationVar(&duration, "duration", 15*time.Minute, "Test time duration")
	flag.IntVar(&archiveCount, "archive-count", 100000, "Archive workflows count")
	flag.IntVar(&archiveCleanSize, "archive-clean-size", 10000, "Archive workflows count of per batch cleaning")
	flag.IntVar(&nodeStatusSize, "node-status-size", 40*1024, "Archive workflows count")
	flag.IntVar(&archiveNodeStatusSize, "archive-node-status-size", 200*1024, "Archive workflows count")
	flag.IntVar(&archiveWorkers, "archive-workers", 10, "Archive workers count")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	conn, err := mysql.ParseURL(dbUri)
	if err != nil {
		log.Fatal(err)
	}

	session, err := mysql.Open(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		session.Close()
	}()
	session.SetMaxOpenConns(5000)
	session.SetMaxIdleConns(150)
	session.SetConnMaxLifetime(60 * time.Second)

	// this is needed to make MySQL run in a Golang-compatible UTF-8 character set.
	_, err = session.SQL().Exec("SET NAMES 'utf8mb4'")
	if err != nil {
		log.Fatal(err)
	}
	_, err = session.SQL().Exec("SET CHARACTER SET utf8mb4")
	if err != nil {
		log.Fatal(err)
	}

	err = sqldb.NewMigrate(session, "local", "argo_workflows").Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}

	instanceIDService := instanceid.NewService("")
	archive := sqldb.NewWorkflowArchive(session, "local", "argo-managed", instanceIDService)

	repo, err := sqldb.NewOffloadNodeStatusRepo(session, "local", "argo_workflows")
	if err != nil {
		log.Fatal(err)
	}

	nodeStatus, err := rand.RandString(nodeStatusSize)
	if err != nil {
		log.Fatal(err)
	}

	archiveNodeStatus, err := rand.RandString(archiveNodeStatusSize)
	if err != nil {
		log.Fatal(err)
	}

	firstStartTime := time.Date(2024, time.January, 1, 00, 0, 0, 0, time.UTC)
	lastStartTime := time.Date(2024, time.January, 1, 00, 0, 0, 0, time.UTC)
	if record, err := getLastArchivedRecord(session); err == nil {
		lastStartTime = record.StartedAt
	}
	log.Infof("Last started workflow time is %s", lastStartTime)

	if count, err := getArchivedWorkflowsCount(session); err == nil {
		log.Infof("There are %d archived workflows currently", count)
	}

	wg := sync.WaitGroup{}
	createdArchiveCount := atomic.Int64{}
	createArchive := func(start, end int) {
		for i := start; i < end; i++ {
			name, err := rand.RandString(16)
			if err != nil {
				log.Warnf("generate workflow name error: %s", err)
				continue
			}
			uid, err := uuid.GenerateUUID()
			if err != nil {
				log.Warnf("generate uuid error: %s", err)
				continue
			}
			offTime := time.Duration(i) * time.Second
			wf := wfv1.Workflow{
				ObjectMeta: v1.ObjectMeta{
					UID:       types.UID(uid),
					Name:      name,
					Namespace: "argo-managed",
					Labels:    map[string]string{},
				},
				Status: wfv1.WorkflowStatus{
					Nodes: map[string]wfv1.NodeStatus{
						"n1": {
							Message: archiveNodeStatus,
						},
					},
					Phase:      wfv1.WorkflowSucceeded,
					StartedAt:  v1.NewTime(lastStartTime.Add(offTime)),
					FinishedAt: v1.NewTime(lastStartTime.Add(offTime)),
				},
			}
			err = archive.ArchiveWorkflow(&wf)
			if err != nil {
				log.Warnf("archive workflow error: %s", err)
				if strings.Contains(err.Error(), "try restarting transaction") {
					i--
				} else {
					continue
				}
			} else {
				createdArchiveCount.Add(1)
			}
		}
		wg.Done()
	}
	workerBatch := archiveCount / archiveWorkers
	for i := 0; i < archiveWorkers; i++ {
		log.Info("Creating archive worker...")
		go createArchive(i*workerBatch, (i+1)*workerBatch)
		wg.Add(1)
	}
	stopPrintCount := make(chan struct{})
	go func(c chan struct{}) {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-c:
				return
			case <-t.C:
				log.Infof("Created archives count is %d", createdArchiveCount.Load())
			}
		}
	}(stopPrintCount)
	wg.Wait()
	stopPrintCount <- struct{}{}
	log.Info("Create archives done")

	if record, err := getFirstArchivedRecord(session); err == nil {
		firstStartTime = record.StartedAt
	}
	log.Infof("First started workflow time is %s", firstStartTime)

	totalUpdate := atomic.Int64{}
	calculateRate := func(startTime time.Time) float64 {
		endTime := time.Now()
		second := endTime.Sub(startTime).Seconds()
		rate := float64(totalUpdate.Load()) / second
		return rate
	}

	reconcileWf := func() {
		uid, err := uuid.GenerateUUID()
		if err != nil {
			log.Fatal(err)
		}
		i := int64(0)
		version := ""
		for {
			select {
			case <-ctx.Done():
				return
			default:
				i++
				if version != "" {
					if _, err = repo.Get(uid, version); err != nil {
						log.Error("get", err)
					}
				}
				time.Sleep(500 * time.Millisecond)
				nodes := map[string]wfv1.NodeStatus{
					"n1": {
						Message: nodeStatus + strconv.FormatInt(i, 10),
					},
				}
				version, err = repo.Save(uid, "argo-managed", nodes)
				if err != nil {
					log.Error("save", err)
				}
				time.Sleep(500 * time.Millisecond)
				totalUpdate.Add(1)
			}
		}
	}

	workflowGarbageCollector := func() {
		defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

		periodicity := envutil.LookupEnvDurationOr("WORKFLOW_GC_PERIOD", 5*time.Minute)
		log.WithField("periodicity", periodicity).Info("Performing periodic GC")
		t := time.NewTicker(periodicity)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				log.Info("Performing periodic workflow GC")
				oldRecords, err := repo.ListOldOffloads("argo-managed")
				if err != nil {
					log.Error(err)
				}
				log.WithField("len_wfs", len(oldRecords)).Info("Deleting old offloads that are not live")
				for uid, versions := range oldRecords {
					for _, version := range versions {
						// skip delete if offload is live
						//log.Info("deleting......")
						if err := repo.Delete(uid, version.Version); err != nil {
							log.Error("delete", err)
						}
					}
				}
				log.Info("Workflow GC finished")
			}
		}
	}

	go workflowGarbageCollector()

	for i := 0; i < parallel; i++ {
		go reconcileWf()
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	stopTimer := time.NewTimer(duration)
	cleanDuration := 3 * time.Minute
	cleanTicker := time.NewTicker(cleanDuration)
	nthCleanBatch := 0
	startTime := time.Now()

	for {
		select {
		case <-cleanTicker.C:
			log.Infof("Average rate is: %f", calculateRate(startTime))

			cleanTicker.Stop()
			totalUpdate.Store(0)
			startTime = time.Now()
			nthCleanBatch += 1
			archiveTTL := time.Now().UTC().Sub(firstStartTime.Add(time.Duration(archiveCleanSize*nthCleanBatch) * time.Second))
			err = archive.DeleteExpiredWorkflows(archiveTTL)
			if err != nil {
				log.Warnf("Clean up expired archive workflows error: %s", err)
			}
			log.Infof("Cleaning %d archives cost %s", archiveCleanSize, time.Since(startTime))
			cleanTicker.Reset(cleanDuration)
			log.Infof("Average rate when cleaning archives is: %f", calculateRate(startTime))

			totalUpdate.Store(0)
			startTime = time.Now()
		case <-stopTimer.C:
			cancel()
			return
		case <-stopCh:
			cancel()
			return
		}
	}
}

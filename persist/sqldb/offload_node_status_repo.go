package sqldb

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

const (
	OffloadNodeStatusDisabled = "Workflow has offloaded nodes, but offloading has been disabled"
	tooLarge                  = "workflow is longer than maximum allowed size for sqldb."

	envVarOffloadMaxSize     = "OFFLOAD_NODE_STATUS_MAX_SIZE"
	envVarCompressNodeStatus = "OFFLOAD_NODE_STATUS_COMPRESSED"
)

type UUIDVersion struct {
	UID     string `db:"uid"`
	Version string `db:"version"`
}

type OffloadNodeStatusRepo interface {
	Save(uid, namespace string, nodes wfv1.Nodes) (string, error)
	Get(uid, version string) (wfv1.Nodes, error)
	List(namespace string) (map[UUIDVersion]wfv1.Nodes, error)
	ListOldOffloads(namespace string) (map[string][]NodesRecord, error)
	Delete(uid, version string) error
	IsEnabled() bool
}

func IsTooLargeError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), tooLarge)
}

func getMaxWorkflowSize() int {
	s, _ := strconv.Atoi(os.Getenv(envVarOffloadMaxSize))
	if s == 0 {
		s = 3 * 1024 * 1024
	}
	return s
}

func NewOffloadNodeStatusRepo(session db.Session, clusterName, tableName string) (OffloadNodeStatusRepo, error) {
	// this environment variable allows you to make Argo Workflows delete offloaded data more or less aggressively,
	// useful for testing
	ttl := env.LookupEnvDurationOr(common.EnvVarOffloadNodeStatusTTL, 5*time.Minute)
	historyCount := env.LookupEnvIntOr(common.EnvVarOffloadNodeStatusHistoryCount, 12)
	log.WithField("ttl", ttl).WithField("historyCount", historyCount).Debug("Node status offloading config")
	return &nodeOffloadRepo{session: session, clusterName: clusterName, tableName: tableName, ttl: ttl, historyCount: historyCount}, nil
}

type nodesRecord struct {
	ClusterName string `db:"clustername"`
	UUIDVersion
	Namespace       string `db:"namespace"`
	Nodes           string `db:"nodes"`
	CompressedNodes string `db:"compressednodes"`
}

type NodesRecord struct {
	UUIDVersion
	Id        int64     `db:"id"`
	UpdatedAt time.Time `db:"updatedat"`
}

type nodeOffloadRepo struct {
	session     db.Session
	clusterName string
	tableName   string
	// time to live - at what ttl an offload becomes old
	ttl time.Duration
	// this number is related to retry number of `reapplyUpdate`
	historyCount int
}

func (wdc *nodeOffloadRepo) IsEnabled() bool {
	return true
}

func nodeStatusVersion(s wfv1.Nodes) (string, string, error) {
	marshalled, err := json.Marshal(s)
	if err != nil {
		return "", "", err
	}

	h := fnv.New32()
	_, _ = h.Write(marshalled)
	return string(marshalled), fmt.Sprintf("fnv:%v", h.Sum32()), nil
}

func (wdc *nodeOffloadRepo) Save(uid, namespace string, nodes wfv1.Nodes) (string, error) {
	marshalled, version, err := nodeStatusVersion(nodes)
	if err != nil {
		return "", err
	}
	raw := marshalled
	size := len(raw)
	compressed := ""
	if os.Getenv(envVarCompressNodeStatus) != "false" {
		raw = "null"
		compressed = file.CompressEncodeString(marshalled)
		size = len(compressed)
	}
	if size > getMaxWorkflowSize() {
		return "", fmt.Errorf("%s compressed size %d > maxSize %d", tooLarge, size, getMaxWorkflowSize())
	}
	record := &nodesRecord{
		ClusterName: wdc.clusterName,
		UUIDVersion: UUIDVersion{
			UID:     uid,
			Version: version,
		},
		Namespace:       namespace,
		Nodes:           raw,
		CompressedNodes: compressed,
	}

	logCtx := log.WithFields(log.Fields{"uid": uid, "version": version})
	logCtx.Debug("Offloading nodes")
	_, err = wdc.session.Collection(wdc.tableName).Insert(record)
	if err != nil {
		// if we have a duplicate, then it must have the same clustername+uid+version, which MUST mean that we
		// have already written this record
		if !isDuplicateKeyError(err) {
			return "", err
		}
		logCtx.WithField("err", err).Info("Ignoring duplicate key error")
	}

	logCtx.Debug("Nodes offloaded, cleaning up old offloads")

	// This might fail, which kind of fine (maybe a bug).
	// It might not delete all records, which is also fine, as we always key on resource version.
	// We also want to keep enough around so that we can service watches.
	var records []NodesRecord
	err = wdc.session.SQL().
		Select("id").
		From(wdc.tableName).
		Where(db.Cond{"clustername": wdc.clusterName}).
		And(db.Cond{"uid": uid}).
		And(db.Cond{"version <>": version}).
		And(wdc.oldOffload()).
		OrderBy("updatedat desc").
		Offset(wdc.historyCount).
		All(&records)
	if err != nil {
		return "", err
	}
	if len(records) > 0 {
		var ids []int64
		for _, r := range records {
			ids = append(ids, r.Id)
		}
		rs, err := wdc.session.SQL().
			DeleteFrom(wdc.tableName).
			Where(db.Cond{"id in": ids}).
			Exec()
		if err != nil {
			return "", err
		}
		rowsAffected, err := rs.RowsAffected()
		if err != nil {
			return "", err
		}
		logCtx.WithField("rowsAffected", rowsAffected).Debug("Deleted offloaded nodes")
	}
	return version, nil
}

func isDuplicateKeyError(err error) bool {
	// postgres
	if strings.Contains(err.Error(), "duplicate key") {
		return true
	}
	// mysql
	if strings.Contains(err.Error(), "Duplicate entry") {
		return true
	}
	return false
}

func (wdc *nodeOffloadRepo) Get(uid, version string) (wfv1.Nodes, error) {
	log.WithFields(log.Fields{"uid": uid, "version": version}).Debug("Getting offloaded nodes")
	r := &nodesRecord{}
	err := wdc.session.SQL().
		SelectFrom(wdc.tableName).
		Where(db.Cond{"clustername": wdc.clusterName}).
		And(db.Cond{"uid": uid}).
		And(db.Cond{"version": version}).
		One(r)
	if err != nil {
		return nil, err
	}
	dbNodes := r.Nodes
	if r.CompressedNodes != "" {
		dbNodes, err = file.DecodeDecompressString(r.CompressedNodes)
		if err != nil {
			return nil, err
		}
	}
	nodes := &wfv1.Nodes{}
	err = json.Unmarshal([]byte(dbNodes), nodes)
	if err != nil {
		return nil, err
	}
	return *nodes, nil
}

func (wdc *nodeOffloadRepo) List(namespace string) (map[UUIDVersion]wfv1.Nodes, error) {
	log.WithFields(log.Fields{"namespace": namespace}).Debug("Listing offloaded nodes")
	var records []nodesRecord
	err := wdc.session.SQL().
		Select("uid", "version", "nodes").
		From(wdc.tableName).
		Where(db.Cond{"clustername": wdc.clusterName}).
		And(namespaceEqual(namespace)).
		All(&records)
	if err != nil {
		return nil, err
	}

	res := make(map[UUIDVersion]wfv1.Nodes)
	for _, r := range records {
		dbNodes := r.Nodes
		if r.CompressedNodes != "" {
			dbNodes, err = file.DecodeDecompressString(r.CompressedNodes)
			if err != nil {
				return nil, err
			}
		}
		nodes := &wfv1.Nodes{}
		err = json.Unmarshal([]byte(dbNodes), nodes)
		if err != nil {
			return nil, err
		}
		res[UUIDVersion{UID: r.UID, Version: r.Version}] = *nodes
	}

	return res, nil
}

func (wdc *nodeOffloadRepo) ListOldOffloads(namespace string) (map[string][]NodesRecord, error) {
	log.WithFields(log.Fields{"namespace": namespace}).Debug("Listing old offloaded nodes")
	var records []NodesRecord
	err := wdc.session.SQL().
		Select("uid", "version", "updatedat").
		From(wdc.tableName).
		Where(db.Cond{"clustername": wdc.clusterName}).
		And(namespaceEqual(namespace)).
		And(wdc.oldOffload()).
		All(&records)
	if err != nil {
		return nil, err
	}
	x := make(map[string][]NodesRecord)
	for _, r := range records {
		x[r.UID] = append(x[r.UID], r)
	}
	return x, nil
}

func (wdc *nodeOffloadRepo) Delete(uid, version string) error {
	if uid == "" {
		return fmt.Errorf("invalid uid")
	}
	if version == "" {
		return fmt.Errorf("invalid version")
	}
	logCtx := log.WithFields(log.Fields{"uid": uid, "version": version})
	logCtx.Debug("Deleting offloaded nodes")
	rs, err := wdc.session.SQL().
		DeleteFrom(wdc.tableName).
		Where(db.Cond{"clustername": wdc.clusterName}).
		And(db.Cond{"uid": uid}).
		And(db.Cond{"version": version}).
		Exec()
	if err != nil {
		return err
	}
	rowsAffected, err := rs.RowsAffected()
	if err != nil {
		return err
	}
	logCtx.WithField("rowsAffected", rowsAffected).Debug("Deleted offloaded nodes")
	return nil
}

func (wdc *nodeOffloadRepo) oldOffload() string {
	return fmt.Sprintf("updatedat < current_timestamp - interval '%d' second", int(wdc.ttl.Seconds()))
}

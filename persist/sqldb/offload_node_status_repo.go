package sqldb

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type OffloadNodeStatusRepo interface {
	Save(name, namespace string, nodes wfv1.Nodes) (string, error)
	Get(name, namespace, version string) (wfv1.Nodes, error)
	List(namespace string) (map[PrimaryKey]wfv1.Nodes, error)
	Delete(name, namespace string) error
	IsEnabled() bool
}

func NewOffloadNodeStatusRepo(tableName string, session sqlbuilder.Database) OffloadNodeStatusRepo {
	return &nodeOffloadRepo{tableName, session}
}

type PrimaryKey struct {
	Name      string `db:"name"`
	Namespace string `db:"namespace"`
	Version   string `db:"version"`
}

type nodesRecord struct {
	PrimaryKey
	Nodes string `db:"nodes"`
}

type nodeOffloadRepo struct {
	tableName string
	session   sqlbuilder.Database
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

func (wdc *nodeOffloadRepo) Save(name, namespace string, nodes wfv1.Nodes) (string, error) {

	marshalled, version, err := nodeStatusVersion(nodes)
	if err != nil {
		return "", err
	}

	record := &nodesRecord{
		PrimaryKey: PrimaryKey{
			Name:      name,
			Namespace: namespace,
			Version:   version,
		},
		Nodes: marshalled,
	}

	logCtx := log.WithFields(log.Fields{"name": name, "namespace": namespace, "version": version})
	logCtx.Debug("Offloading nodes")
	_, err = wdc.session.Collection(wdc.tableName).Insert(record)
	if err != nil {
		// if we have a duplicate, then it must have the same name+namespace+offloadVersion, which MUST mean that we
		// have already written this record
		if !strings.Contains(err.Error(), "duplicate key") {
			return "", err
		}
	}

	logCtx.Info("Nodes offloaded, cleaning up old offloads")

	// This might fail, which kind of fine (maybe a bug).
	// It might not delete all records, which is also fine, as we always key on resource version.
	// We also want to keep enough around so that we can service watches.
	_, err = wdc.session.
		DeleteFrom(wdc.tableName).
		Where(db.Cond{"name": name}).
		And(db.Cond{"namespace": namespace}).
		And(db.Cond{"version <>": version}).
		// TODO - probably needs to be shorter - but how short?
		And(db.Cond{"updatedat + interval '10' second <": "now()"}).
		Exec()
	if err != nil {
		return "", err
	}
	return version, nil
}

func (wdc *nodeOffloadRepo) Get(name, namespace, version string) (wfv1.Nodes, error) {
	log.WithFields(log.Fields{"name": name, "namespace": namespace, "version": version}).Debug("Getting offloaded nodes")
	r := &nodesRecord{}
	err := wdc.session.
		Select("nodes").
		From(wdc.tableName).
		Where(db.Cond{"name": name}).
		And(db.Cond{"namespace": namespace}).
		And(db.Cond{"version": version}).
		One(r)
	if err != nil {
		return nil, err
	}
	nodes := &wfv1.Nodes{}
	err = json.Unmarshal([]byte(r.Nodes), nodes)
	if err != nil {
		return nil, err
	}
	return *nodes, nil
}

func (wdc *nodeOffloadRepo) List(namespace string) (map[PrimaryKey]wfv1.Nodes, error) {
	log.WithFields(log.Fields{"namespace": namespace}).Debug("Listing offloaded nodes")
	var records []nodesRecord
	err := wdc.session.
		Collection(wdc.tableName).
		Find(namespaceEqual(namespace)).
		All(&records)
	if err != nil {
		return nil, err
	}

	res := make(map[PrimaryKey]wfv1.Nodes)
	for _, r := range records {
		nodes := &wfv1.Nodes{}
		err = json.Unmarshal([]byte(r.Nodes), nodes)
		if err != nil {
			return nil, err
		}
		res[r.PrimaryKey] = *nodes
	}

	return res, nil
}

func (wdc *nodeOffloadRepo) Delete(name, namespace string) error {
	log.WithFields(log.Fields{"name": name, "namespace": namespace}).Debug("Deleting offloaded node")
	return wdc.session.Collection(wdc.tableName).Find(db.Cond{"name": name}).And(db.Cond{"namespace": namespace}).Delete()
}

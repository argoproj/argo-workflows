package fixtures

import (
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/postgresql"
)

type Persistence struct {
	session sqlbuilder.Database
}

func newPersistence() *Persistence {
	session, err := postgresql.Open(postgresql.ConnectionURL{User: "postgres", Password: "password", Host: "localhost"})
	if err != nil {
		panic(err)
	}
	return &Persistence{session}
}

func (s *Persistence) OffloadedCount() int {
	count, err := s.session.Collection("argo_workflows").Find().Count()
	if err != nil {
		panic(err)
	}
	return int(count)
}

func (s *Persistence) Close() error {
	return s.session.Close()
}

func (s *Persistence) DeleteEverything() {
	_, err := s.session.DeleteFrom("argo_workflows").Exec()
	if err != nil {
		panic(err)
	}
	_, err = s.session.DeleteFrom("argo_archived_workflows").Exec()
	if err != nil {
		panic(err)
	}
}

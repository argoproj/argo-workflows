package persist

type UUIDVersion struct {
	UID     string `db:"uid"`
	Version string `db:"version"`
}

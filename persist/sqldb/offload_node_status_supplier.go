package sqldb

type OffloadNodeStatusRepoSupplier interface {
	GetOffloadNodeStatusRepo() OffloadNodeStatusRepo
}

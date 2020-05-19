package rbac

type Interface interface {
	ServiceAccount(groups []string) (string, error)
}

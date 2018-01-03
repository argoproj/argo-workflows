package hack

import (
	// The following is a dummy import to coerce dep to believe we have a dependency on
	// the k8s.io/code-generator repository so that it will vendor and install it during
	// a `dep ensure`
	_ "k8s.io/code-generator/cmd/client-gen/generators"
)

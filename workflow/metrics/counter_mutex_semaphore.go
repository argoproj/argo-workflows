package metrics

import (
	"context"
	"strings"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func (m *Metrics) AddMutex(ctx context.Context, name, namespace string) {
    if m == nil || m.Metrics == nil {
        return
    }
    m.AddInt(ctx, telemetry.InstrumentMutexTotal.Name(), 1, telemetry.InstAttribs{
        {Name: telemetry.AttribMutexName, Value: name},
        {Name: telemetry.AttribMutexNamespace, Value: namespace},
    })
}

func (m *Metrics) RemoveMutex(ctx context.Context, name, namespace string) {
    if m == nil || m.Metrics == nil {
        return
    }
    m.AddInt(ctx, telemetry.InstrumentMutexTotal.Name(), -1, telemetry.InstAttribs{
        {Name: telemetry.AttribMutexName, Value: name},
        {Name: telemetry.AttribMutexNamespace, Value: namespace},
    })
}

func (m *Metrics) AddSemaphore(ctx context.Context, name, namespace string) {
    if m == nil || m.Metrics == nil {
        return
    }
    cmName, semName := parseSemaphoreLockName(name)
    attribs := telemetry.InstAttribs{
        {Name: telemetry.AttribSemaphoreConfigMapName, Value: cmName},
        {Name: telemetry.AttribSemaphoreName, Value: semName},
        {Name: telemetry.AttribSemaphoreNamespace, Value: namespace},
    }
    m.AddInt(ctx, telemetry.InstrumentSemaphoreTotal.Name(), 1, attribs)
}

func (m *Metrics) RemoveSemaphore(ctx context.Context, name, namespace string) {
    if m == nil || m.Metrics == nil {
        return
    }
    cmName, semName := parseSemaphoreLockName(name)
    attribs := telemetry.InstAttribs{
        {Name: telemetry.AttribSemaphoreConfigMapName, Value: cmName},
        {Name: telemetry.AttribSemaphoreName, Value: semName},
        {Name: telemetry.AttribSemaphoreNamespace, Value: namespace},
    }
    m.AddInt(ctx, telemetry.InstrumentSemaphoreTotal.Name(), -1, attribs)
}

// parseSemaphoreLockName tries to derive configmap and semaphore logical names from encoded lock key forms.
// Supported patterns (examples):
// default/ConfigMap/my-config-semaphore  -> (my-config-semaphore, my-config-semaphore)
// default/Database/my-db-semaphore      -> (my-db-semaphore, my-db-semaphore)
// default/Semaphore/my-sem              -> (my-sem, my-sem)
// If pattern unrecognized returns (name,name).
func parseSemaphoreLockName(full string) (string, string) {
    // We expect segments split by '/'. We only care about last segment as logical semaphore name.
    // ConfigMap form has at least 3 segments after namespace: Namespace/ConfigMap/<cm-name>
    parts := strings.Split(full, "/")
    if len(parts) < 2 { // minimal sanity
        return full, full
    }
    // last part always the logical semaphore name.
    semName := parts[len(parts)-1]
    // If second element is ConfigMap or Database, we take last as both cmName and semName.
    if len(parts) >= 3 {
        switch parts[1] { // parts[0]=namespace, parts[1]=Kind
        case "ConfigMap", "Database", "Semaphore":
            return semName, semName
        }
    }
    return semName, semName
}

package workflow

import (
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/fields"
)

// applyTemplates puts a hydrated template set onto the workflow object for the API response.
func applyTemplates(wf *wfv1.Workflow, templates []wfv1.Template) {
	wf.Spec.Templates = templates
	if wf.Status.StoredTemplates == nil {
		wf.Status.StoredTemplates = make(map[string]wfv1.Template, len(templates))
	}
	for _, tmpl := range templates {
		wf.Status.StoredTemplates[tmpl.Name] = tmpl
	}
}

// shouldHydrateTemplates reports whether the cleaned response will carry template data.
// Watch clients that request an explicit field list without templates (the UI list watch)
// would otherwise pay a database read and unmarshal for every event only to have the
// cleaner drop the result.
func shouldHydrateTemplates(cleaner fields.Cleaner) bool {
	return !cleaner.WillExclude("spec.templates") ||
		!cleaner.WillExclude("status.storedTemplates") ||
		!cleaner.WillExclude("status.storedTemplateSpecs")
}

// maxHydrationMemoEntries bounds one stream's memo so a broad watch cannot grow memory
// without limit. A stream that exceeds it falls back to hydrating each event.
const maxHydrationMemoEntries = 16

type hydrationMemoEntry struct {
	version   string
	templates []wfv1.Template
}

// hydrationMemo caches the templates hydrated during one watch stream, keyed by workflow UID.
// Templates are immutable for a (uid, StoredTemplateSpecs.Version), and the stream sends the
// full object on every event, so one database read per version is enough.
type hydrationMemo struct {
	entries map[types.UID]hydrationMemoEntry
}

func newHydrationMemo() *hydrationMemo {
	return &hydrationMemo{entries: make(map[types.UID]hydrationMemoEntry)}
}

// get returns the memoized templates for the workflow's current offload version.
func (m *hydrationMemo) get(wf *wfv1.Workflow) ([]wfv1.Template, bool) {
	marker := wf.Status.StoredTemplateSpecs
	if marker == nil || marker.Version == "" {
		return nil, false
	}
	e, ok := m.entries[wf.UID]
	if !ok || e.version != marker.Version {
		return nil, false
	}
	return e.templates, true
}

// put memoizes templates under the version they were hydrated for.
func (m *hydrationMemo) put(wf *wfv1.Workflow, version string, templates []wfv1.Template) {
	if version == "" || len(templates) == 0 {
		return
	}
	if _, ok := m.entries[wf.UID]; !ok && len(m.entries) >= maxHydrationMemoEntries {
		// Arbitrary eviction: the memo is an optimization, and entries are small in number.
		for uid := range m.entries {
			delete(m.entries, uid)
			break
		}
	}
	m.entries[wf.UID] = hydrationMemoEntry{version: version, templates: templates}
}

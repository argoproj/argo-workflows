Description: Autocomplete the Namespace filter from the namespaces of resources currently visible on the page
Author: [Morgan Allen](https://github.com/callmemorgan)
Component: UI
Issues: 7405

The Namespace filter on the workflows, workflow templates, cron workflows, sensors, event sources, event bindings, and event flow pages now autocompletes from the namespaces of resources already loaded for that page, in addition to the existing localStorage history.

When the user is viewing resources across multiple namespaces (cluster-wide mode), the namespace filter dropdown now lists the namespaces present on the current page and narrows as you type. No new server endpoint or RBAC is involved — suggestions are derived from data the user already has permission to see.

The managed-namespace short-circuit (where the filter renders as plain text) is unchanged.

Description: Add pendingTimeout field to templates for setting maximum time in pending status.
Authors: [Dennis Lawler](https://github.com/drawlerr)
Component: General
Issues: 10341

Adds a new `pendingTimeout` field to workflow templates that allows setting a maximum duration a pod can spend in Pending status.
This is useful when pods may be stuck pending due to resource constraints, scheduling issues, or node availability.
Unlike the existing `timeout` field which covers the entire node lifecycle, `pendingTimeout` specifically targets the pending phase.

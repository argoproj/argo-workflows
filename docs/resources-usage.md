# Resource Usage

> v2.13 and after

Argo Workflows provides an indication of how much  resource your workflow has used and saves this 
information. This is intended to be an **indicative but not accurate** value.

This is taken just once during the lifetime of each pod, just after it is running, rather than sampled over time (which would give a much more accurate figure).

# Continuous Integration Example

Continuous integration is a popular application for workflows. Currently, Argo does not provide event triggers for automatically kicking off your CI jobs, but we plan to do so in the near future. Until then, you can easily write a cron job that checks for new commits and kicks off the needed workflow, or use your existing Jenkins server to kick off the workflow.

A good example of a CI workflow spec is provided at <https://github.com/argoproj/argo-workflows/tree/master/examples/influxdb-ci.yaml>. Because it just uses the concepts that we've already covered and is somewhat long, we don't go into details here.

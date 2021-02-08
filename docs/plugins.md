# Plugins

You can extend the capabilities of workflows by writing a plugin. Typical use cases are:

* Sending a notification, e.g. Slack.
* Starting a Spark job.
* Sending a report to a database.  
* Interop with Kafka, AirFlow, TektonCD, Argo CD etc
* Sending messages to in-house tooling.

# Writing a Plugin

You have two options:

* Write a JSON-RPC service. Easy, can be witting any programming language. 
* Write a Golang plugin. Complex, high maintenance, but 1st-class and highly-scalable.

j
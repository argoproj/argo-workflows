set -eu

kubectl create ns argo-events

kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

# clean-up

kubectl -n argo-events delete sensor,es,wf --all && killall kubectl

# demo 1

kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/calendar.yaml
kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml
kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/multi-dependencies.yaml

kubectl -n argo-events port-forward "$(kubectl -n argo-events get pod -l eventsource-name=webhook -o name)" 12000:12000 &

curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

kubectl -n argo-events get wf

# demo 2 - parameterization
# https://argoproj.github.io/argo-events/tutorials/02-parameterization/
kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml
kubectl -n argo-events port-forward "$(kubectl -n argo-events get pod -l eventsource-name=webhook -o name)" 12000:12000 &

kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/02-parameterization/sensor-01.yaml

curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

kubectl -n argo-events get wf

# demo 3 - conditions

# kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/06-trigger-conditions/webhook-event-source.yaml
kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml
kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/06-trigger-conditions/minio-event-source.yaml
kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/06-trigger-conditions/sensor-01.yaml

kubectl -n argo-events port-forward "$(kubectl -n argo-events get pod -l eventsource-name=webhook -o name)" 12000:12000 &
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

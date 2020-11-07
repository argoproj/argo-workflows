set -eu

echo "Installing Argo Events"
kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/test/manifests/argo-events-ns.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

echo "Creating example set-up"
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/calendar.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/webhook.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/multi-dependencies.yaml

echo "Waiting for pods to be ready"
kubectl -n argo-events wait --for=condition=Ready pod --all

echo "Port-forwarding webhook"
kubectl -n argo-events port-forward "$(kubectl -n argo-events get pod -l eventsource-name=webhook -o name)" 12000:12000 &
sleep 2s

echo "Sending test message"
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
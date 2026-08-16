#!/usr/bin/env bash
set -euo pipefail

# Create service account and token
kubectl create serviceaccount argo-sdk-test -n argo 2>/dev/null || true
kubectl create rolebinding argo-sdk-test --clusterrole=argo-server-cluster-role --serviceaccount=argo:argo-sdk-test -n argo 2>/dev/null || true
export ARGO_TOKEN="Bearer $(kubectl create token argo-sdk-test -n argo --duration=1h)"

echo "Running Go SDK examples..."

# Declare associative array for test configurations
# Each key is the example directory name, value is a list of parameter sets separated by '|'
# Each parameter set can contain multiple flags
declare -A test_configs=(
	["grpc-client"]="--insecure-skip-verify"
)

# Examples to skip (e.g., require specific runtime environment)
declare -A skip_examples=(
    ["alternate_auth"]="Requires in-cluster environment or service account tokens"
)

for dir in sdks/go/*/; do
    if [ -f "$dir/go.mod" ]; then
        example_name=$(basename "$dir")

        # Check if this example should be skipped
        if [[ -n "${skip_examples[$example_name]:-}" ]]; then
            echo "Skipping $dir: ${skip_examples[$example_name]}"
            continue
        fi

        # Check if this example has custom configurations
        if [[ -n "${test_configs[$example_name]:-}" ]]; then
            # Split configurations by '|' and run each one
            IFS='|' read -ra configs <<< "${test_configs[$example_name]}"
            for config in "${configs[@]}"; do
                echo "Running $dir with params: $config..."
                (cd "$dir" && go get . && go run . $config) || exit 1
            done
        else
            # Run with no parameters
            echo "Running $dir..."
            (cd "$dir" && go get . && go run .) || exit 1
        fi
    fi
done

echo "All Go SDK examples ran successfully"

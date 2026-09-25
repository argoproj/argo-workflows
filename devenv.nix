{ pkgs, lib, config, inputs, ... }:

let
  # Access packages from the argo-flake input
  argoFlakePackages = inputs.argo-flake.packages.${pkgs.stdenv.hostPlatform.system};

  argoConfig = import ./dev/nix/conf.nix;
in
{
  # Import packages from nixpkgs and the argo flake
  packages = with argoFlakePackages; [
    go
    nodejs
    yarn
    jq
    protobuf
    diffutils
    kubeauto
    mockery
    protoc-gen-gogo-all
    grpc-ecosystem
    go-swagger
    controller-tools
    k8sio-tools
    goreman
    stern
    buf
    nodeDependencies
    golangci-lint
    typos
    cspell
  ] ++ [
    # The dev stack now runs in-cluster via Tilt (replacing the old host-process
    # flow). nixpkgs' tilt is 0.37.3 — the same version the Makefile pins — so
    # `make tilt`/`make k3d` become no-ops. k3d also comes from nixpkgs here so
    # the environment is self-contained.
    pkgs.tilt
    pkgs.k3d
    pkgs.kubectl
  ];

  # Set up environment
  env = argoConfig.env // {
    USE_NIX = "true";
  };

  # Create the k3d cluster Tilt deploys into (context k3d-k3s-default), using the
  # same script `make start` runs. Ordered before the tilt process so the cluster
  # exists when Tilt connects.
  tasks."argo:cluster" = {
    exec = "make k3d-up";
    before = [ "devenv:processes:tilt" ];
  };

  # Tilt builds the controller/server/executor images, runs them in-cluster, runs
  # the UI dev server (hot-reload on :8080), and sets up the port-forwards —
  # replacing the old host processes (kubeauto, workflow-controller, argo-server,
  # ui) and `make install`. `--stream` prints build/pod logs to stdout instead of
  # opening the TUI, which suits devenv's process-compose supervisor. Run tilt
  # directly (not `make start`) to use nixpkgs' tilt/k3d rather than the Makefile's
  # download-to-GOPATH install targets.
  processes = {
    tilt.exec = "tilt up --stream --host=0.0.0.0 -- --profile=minimal";
  };
  enterShell = ''
    # --- 1. Environment Guard & Local Paths ---
    # We isolate the Go environment into the project folder to keep Nix pure.
    unset GOPATH GOROOT
    mkdir -p .go/src .gocache .goenv .devenv

    export GOPATH="$PWD/.go"
    export GOCACHE="$PWD/.gocache"
    export GOENV="$PWD/.goenv"
    # Add the local bin to PATH so installed tools are available
    export PATH="$GOPATH/bin:$PATH"

    # --- 2. Port Cleanup ---
    # Ensures a clean state for the UI and API servers
    ./hack/free-port.sh 9090
    ./hack/free-port.sh 2746
    ./hack/free-port.sh 8080

    # --- 3. Smart Sync: Populate .go/src (Source Only) ---
    # Triggered if go.mod or devenv.nix changes.
    if [ ! -f .devenv/go_synced ] || [ go.mod -nt .devenv/go_synced ] || [ devenv.nix -nt .devenv/go_synced ]; then
      echo "📥 Populating .go/src with dependency source code..."

      MODULES=(
        "sigs.k8s.io/controller-tools@v0.18.0"
        "k8s.io/code-generator@v0.33.1"
        "github.com/gogo/protobuf@v1.3.2"
        "github.com/grpc-ecosystem/grpc-gateway@v1.16.0"
        "k8s.io/kube-openapi@424119656bbf"
      )

      export GOFLAGS="-mod=mod"
      for MOD in "''${MODULES[@]}"; do
        # Ensure module is in cache
        go mod download "$MOD"

        # Resolve the literal directory in the Nix/Go cache
        MOD_NAME=$(echo "$MOD" | cut -d'@' -f1)
        CACHE_PATH=$(go list -m -f '{{.Dir}}' "$MOD")

        if [ -n "$CACHE_PATH" ] && [ -d "$CACHE_PATH" ]; then
          TARGET_PATH="$GOPATH/src/$MOD_NAME"
          mkdir -p "$(dirname "$TARGET_PATH")"

          # Create a symbolic link farm (fast, space-efficient)
          rm -rf "$TARGET_PATH"
          cp -as "$CACHE_PATH" "$TARGET_PATH"

          # Reset permissions (Nix/Go cache is read-only 0444)
          chmod -R +w "$TARGET_PATH"
        else
          echo "⚠️  Failed to locate source for $MOD"
        fi
      done
      unset GOFLAGS;

      touch .devenv/go_synced
    fi

    # --- 4. Smart Sync: UI Dependencies ---
    # Triggered if ui/package.json changes.
    if [ -d "ui" ]; then
      if [ ! -f .devenv/ui_synced ] || [ ui/package.json -nt .devenv/ui_synced ]; then
        echo "🎨 UI dependencies changed. Running yarn install..."
        yarn --cwd ui install
        touch .devenv/ui_synced
      fi
    fi

    # Manifests and images are applied by Tilt (`devenv up`), not `make install`.

    echo "-------------------------------------------------------"
    echo "✅ Environment Ready — run 'devenv up' to start the Tilt dev stack"
    echo "GOPATH: $GOPATH"
    echo "Source: $(ls .go/src | xargs)"
    echo "-------------------------------------------------------"
  '';
}

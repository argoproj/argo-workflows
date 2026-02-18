{ pkgs, lib, config, inputs, ... }:

let
  # Access packages from the argo-flake input
  argoFlakePackages = inputs.argo-flake.packages.${pkgs.stdenv.hostPlatform.system};
  
  argoConfig = import ./dev/nix/conf.nix;
  
  mkEnvSerialize = (envKey: envValue: "export ${envKey}=${envValue};");
  mkEnv = (envAttrs:
    lib.concatStrings
      (lib.mapAttrsToList
        mkEnvSerialize
        envAttrs)
  );
  mkExec = (execName: envAttrs: execArgs:
    "${mkEnv envAttrs}${execName} ${execArgs}"
  );
  controllerCmd = mkExec "./dist/workflow-controller" argoConfig.controller.env argoConfig.controller.args;
  argoServerCmd = mkExec "./argo" argoConfig.argoServer.env argoConfig.argoServer.args;
  uiCmd = mkExec "yarn" argoConfig.ui.env argoConfig.ui.args;
in
{
  # Import packages from nixpkgs and the argo flake
  packages = with pkgs; [
    go_1_25
    nodejs_20
    yarn
    jq
    protobuf
    diffutils
    argoFlakePackages.kubeauto  # Import kubeauto from the argo flake
    argoFlakePackages.mockery
    argoFlakePackages.protoc-gen-gogo-all
    argoFlakePackages.grpc-ecosystem
    argoFlakePackages.go-swagger
    argoFlakePackages.controller-tools
    argoFlakePackages.k8sio-tools
    argoFlakePackages.goreman
    argoFlakePackages.stern
    argoFlakePackages.buf
    argoFlakePackages.nodeDependencies
    golangci-lint
  ];

  # Set up environment
  env = argoConfig.env // {
    USE_NIX = "true";
  };

  # Define processes directly. 
  # We removed the explicit 'process.managers.process-compose.settings' dependency graph 
  # to avoid bugs/deadlocks. devenv will start these in parallel using its native process-compose backend.
  processes = {
    kubeauto = {
      exec = "kubeauto";
    };
    workflow-controller = {
      exec = controllerCmd;
    };
    argo-server = {
      exec = argoServerCmd;
    };
    ui = {
      exec = uiCmd;
    };
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

    # --- 5. Smart Sync: Manifests & App Source ---
    # Runs 'make install' if any local code or manifests are updated.
    SEARCH_PATHS=""
    [ -d "src" ] && SEARCH_PATHS="$SEARCH_PATHS src"
    [ -d "manifests" ] && SEARCH_PATHS="$SEARCH_PATHS manifests"
    [ -d "api" ] && SEARCH_PATHS="$SEARCH_PATHS api"

    if [ -n "$SEARCH_PATHS" ]; then
      LATEST_CHANGE=$(find $SEARCH_PATHS -type f -printf '%T@ %p\n' 2>/dev/null | sort -n | tail -1 | cut -f2- -d" ")
      
      if [ ! -f .devenv/make_installed ] || [ "$LATEST_CHANGE" -nt .devenv/make_installed ]; then
        echo "🛠️  Local source or manifests changed. Running make install..."
        make install PROFILE=minimal
        touch .devenv/make_installed
      fi
    fi

    echo "-------------------------------------------------------"
    echo "✅ Environment Ready"
    echo "GOPATH: $GOPATH"
    echo "Source: $(ls .go/src | xargs)"
    echo "-------------------------------------------------------"
  '';
}

{ pkgs, lib, config, inputs, ... }:

let
  # Access packages from the argo-flake input
  argoFlakePackages = inputs.argo-flake.packages.${pkgs.system};
  
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
    go
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
  ];

  # Set up environment
  env = argoConfig.env;

  # Define processes with dependencies
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

  # Configure process dependencies
  process.managers.process-compose.settings = {
    processes = {
      workflow-controller = {
        depends_on = {
          kubeauto = {
            condition = "process_started";
          };
        };
      };
      argo-server = {
        depends_on = {
          kubeauto = {
            condition = "process_started";
          };
        };
      };
      ui = {
        depends_on = {
          kubeauto = {
            condition = "process_started";
          };
        };
      };
    };
  };

  enterShell = ''
    unset GOPATH;
    unset GOROOT;
    ./hack/free-port.sh 9090;
    ./hack/free-port.sh 2746;
    ./hack/free-port.sh 8080;
    yarn --cwd ui install;
    sleep 5;
    clear;
    make install PROFILE=minimal
  '';
} 

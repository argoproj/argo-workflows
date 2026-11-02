{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-22.11";
    flake-parts = { url = "github:hercules-ci/flake-parts"; inputs.nixpkgs-lib.follows = "nixpkgs"; };
    nix-filter = { url = "github:numtide/nix-filter"; };
    treefmt-nix.url = "github:numtide/treefmt-nix";
  };

  outputs = inputs:
    inputs.flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      imports = [ inputs.treefmt-nix.flakeModule ];
      perSystem = { pkgs, lib, config, ... }:
        let

          openapi_generator_cli_5_2_1 = pkgs.openapi-generator-cli.overrideAttrs (oldAttrs: rec {
            pname = "openapi-generator-cli";
            version = "5.2.1"; # update this when updating sdk Makefile
            jarfilename = "${pname}-${version}.jar";
            src = pkgs.fetchurl {
              url = "mirror://maven/org/openapitools/${pname}/${version}/${jarfilename}";
              sha256 = "sha256-stRtSZCvPUQuTiKOHmJ7k8o3Gtly9Up+gicrDOeWjIs=";
            };
          });
          argoConfig = import ../../dev/nix/conf.nix;
          filter = inputs.nix-filter.lib;
          mysrc = filter {
            root = ./../../.;
            include = [
              "sdks/java/settings.xml"
              "api/openapi-spec/swagger.json"
            ];
            exclude = [
              "Makefile"
            ];
          };
        in
        {
          packages = rec {
            argo_workflows_java_sdk = pkgs.stdenv.mkDerivation rec {
              version = argoConfig.version;
              pname = "argo-client-java-${version}";

              nativeBuildInputs = [
                openapi_generator_cli_5_2_1
                pkgs.gnused
                pkgs.openjdk8-bootstrap
              ];

              src = mysrc;

              buildPhase = ''
                pushd sdks/java
                export WD=$(echo `pwd`/client)
                mkdir -p $WD 
                cp settings.xml $WD/settings.xml
                cat ../../api/openapi-spec/swagger.json | sed 's/io.k8s.api.core.v1.//' | sed 's/io.k8s.apimachinery.pkg.apis.meta.v1.//' > $WD/swagger.json
                export GIT_TAG=$(git describe --exact-match --tags --abbrev=0 2> /dev/null || echo untagged)
                if [ "$GIT_TAG" == "untagged" ]; then
                  export VERSION=0.0.0-SNAPSHOT
                else
                  export VERSION=$(echo "$GIT_TAG" | sed -e "s/^v//")
                fi
                mkdir -p $out/data
                openapi-generator-cli generate -i client/swagger.json -g java -o ./client -p hideGenerationTimestamp=true -p serializationLibrary=jsonb -p dateLibrary=java8 --api-package io.argoproj.workflow.apis --invoker-package io.argoproj.workflow --model-package io.argoproj.workflow.models --skip-validate-spec --group-id io.argoproj.workflow --artifact-id argo-client-java --artifact-version $VERSION --import-mappings Time=java.time.Instant --import-mappings Affinity=io.kubernetes.client.openapi.models.V1Affinity --import-mappings ConfigMapKeySelector=io.kubernetes.client.openapi.models.V1ConfigMapKeySelector --import-mappings Container=io.kubernetes.client.openapi.models.V1Container --import-mappings ContainerPort=io.kubernetes.client.openapi.models.V1ContainerPort --import-mappings EnvFromSource=io.kubernetes.client.openapi.models.V1EnvFromSource --import-mappings EnvVar=io.kubernetes.client.openapi.models.V1EnvVar --import-mappings HostAlias=io.kubernetes.client.openapi.models.V1HostAlias --import-mappings Lifecycle=io.kubernetes.client.openapi.models.V1Lifecycle --import-mappings ListMeta=io.kubernetes.client.openapi.models.V1ListMeta --import-mappings LocalObjectReference=io.kubernetes.client.openapi.models.V1LocalObjectReference --import-mappings ObjectMeta=io.kubernetes.client.openapi.models.V1ObjectMeta --import-mappings ObjectReference=io.kubernetes.client.openapi.models.V1ObjectReference --import-mappings PersistentVolumeClaim=io.kubernetes.client.openapi.models.V1PersistentVolumeClaim --import-mappings PodDisruptionBudgetSpec=io.kubernetes.client.openapi.models.V1beta1PodDisruptionBudgetSpec --import-mappings PodDNSConfig=io.kubernetes.client.openapi.models.V1PodDNSConfig --import-mappings PodSecurityContext=io.kubernetes.client.openapi.models.V1PodSecurityContext --import-mappings Probe=io.kubernetes.client.openapi.models.V1Probe --import-mappings ResourceRequirements=io.kubernetes.client.openapi.models.V1ResourceRequirements --import-mappings SecretKeySelector=io.kubernetes.client.openapi.models.V1SecretKeySelector --import-mappings SecurityContext=io.kubernetes.client.openapi.models.V1SecurityContext --import-mappings Toleration=io.kubernetes.client.openapi.models.V1Toleration --import-mappings Volume=io.kubernetes.client.openapi.models.V1Volume --import-mappings VolumeDevice=io.kubernetes.client.openapi.models.V1VolumeDevice --import-mappings VolumeMount=io.kubernetes.client.openapi.models.V1VolumeMount --generate-alias-as-model
                pushd client 
                sed 's/<dependencies>/<dependencies><dependency><groupId>io.kubernetes<\/groupId><artifactId>client-java<\/artifactId><version>14.0.1<\/version><\/dependency>/g' pom.xml > tmp && mv tmp pom.xml
                popd

                popd
              '';

              installPhase = ''
                pushd sdks/java
                mkdir -p $out/data
                cp ./client/swagger.json $out/data/swagger.json
                cp -r ./client $out/data
                popd
              '';
            };
            default = argo_workflows_java_sdk;
          };

          devShells = {
            default = pkgs.mkShell {
              packages = with pkgs; [
                openapi_generator_cli_5_2_1
                openjdk8-bootstrap
                gnused
              ];
            };
          };

          treefmt = {
            projectRootFile = "flake.nix";
            programs.nixpkgs-fmt.enable = true;
          };
        };
    };

}

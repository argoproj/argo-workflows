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

          openapi_generator_cli_5_4_0 = pkgs.openapi-generator-cli.overrideAttrs (oldAttrs: rec {
            pname = "openapi-generator-cli";
            version = "5.4.0"; # update this when updating sdk Makefile
            jarfilename = "${pname}-${version}.jar";
            src = pkgs.fetchurl {
              url = "mirror://maven/org/openapitools/${pname}/${version}/${jarfilename}";
              sha256 = "sha256-8+0xIxDjkDJLM7ov//KQzoEpNSB6FJPsXAmNCkQb5Rw=";
            };
          });
          pythonEnv = pkgs.python310.withPackages (ps: [
            ps.pytest
            ps.typing-extensions
            ps.mypy
            ps.autopep8
            ps.pip
            ps.build
            ps.twine
            ps.setuptools
          ]);
          argoConfig = import ../../conf.nix;
          filter = inputs.nix-filter.lib;
          mysrc = filter {
            root = ./../../.;
            include = [
              "api/openapi-spec/swagger.json"
              "sdks/python/sdk_version.py"
              "hack/custom-boilerplate.go.txt"
              "LICENSE"
            ];
            exclude = [
              "Makefile"
            ];
          };
        in
        {
          packages = rec {
            argo_workflows_python_sdk = pkgs.stdenv.mkDerivation rec {
              version = argoConfig.version;
              pname = "argo-client-python-${version}";

              nativeBuildInputs = [
                openapi_generator_cli_5_4_0
                pkgs.gnused
                pythonEnv
              ];

              src = mysrc;

              buildPhase = ''
                pushd sdks/python
                export WD=$(echo `pwd`/client)
                mkdir -p $WD 
                cat ../../api/openapi-spec/swagger.json | sed 's/io.k8s.api.core.v1.//' | sed 's/io.k8s.apimachinery.pkg.apis.meta.v1.//' > $WD/swagger.json
                cp ../../LICENSE $WD/LICENSE
                export VERSION=$(./sdk_version.py)
                openapi-generator-cli generate --input-spec ./client/swagger.json --generator-name python --output ./client --additional-properties packageVersion=$VERSION --additional-properties packageName="argo_workflows" --additional-properties projectName="argo-workflows" --additional-properties hideGenerationTimestamp=true --remove-operation-id-prefix --model-name-prefix "" --model-name-suffix "" --artifact-id argo-python-client --global-property modelTests=false --global-property packageName=argo_workflows --generate-alias-as-model
                popd
              '';

              installPhase = ''
                pushd sdks/python
                mkdir -p $out/data
                cp -r client $out/data/
                popd
              '';
            };
            default = argo_workflows_python_sdk;
          };

          devShells = {
            default = pkgs.mkShell {
              packages = with pkgs; [
                openapi_generator_cli_5_4_0
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

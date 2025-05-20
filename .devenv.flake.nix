{
  inputs =
    let
      version = "1.6.1";
system = "x86_64-linux";
devenv_root = "/home/isubasinghe/go/src/github.com/argoproj/argo-workflows";
devenv_dotfile = ./.devenv;
devenv_dotfile_string = ".devenv";
container_name = null;
devenv_tmpdir = "/run/user/1000";
devenv_runtime = "/run/user/1000/devenv-d0596fb";
devenv_istesting = false;
devenv_direnvrc_latest_version = 1;

        in {
        git-hooks.url = "github:cachix/git-hooks.nix";
      git-hooks.inputs.nixpkgs.follows = "nixpkgs";
      pre-commit-hooks.follows = "git-hooks";
      nixpkgs.url = "github:cachix/devenv-nixpkgs/rolling";
      devenv.url = "github:cachix/devenv?dir=src/modules";
      } // (if builtins.pathExists (devenv_dotfile + "/flake.json")
      then builtins.fromJSON (builtins.readFile (devenv_dotfile +  "/flake.json"))
      else { });

      outputs = { nixpkgs, ... }@inputs:
        let
          version = "1.6.1";
system = "x86_64-linux";
devenv_root = "/home/isubasinghe/go/src/github.com/argoproj/argo-workflows";
devenv_dotfile = ./.devenv;
devenv_dotfile_string = ".devenv";
container_name = null;
devenv_tmpdir = "/run/user/1000";
devenv_runtime = "/run/user/1000/devenv-d0596fb";
devenv_istesting = false;
devenv_direnvrc_latest_version = 1;

            devenv =
            if builtins.pathExists (devenv_dotfile + "/devenv.json")
            then builtins.fromJSON (builtins.readFile (devenv_dotfile + "/devenv.json"))
            else { };
          getOverlays = inputName: inputAttrs:
            map
              (overlay:
                let
                  input = inputs.${inputName} or (throw "No such input `${inputName}` while trying to configure overlays.");
                in
                  input.overlays.${overlay} or (throw "Input `${inputName}` has no overlay called `${overlay}`. Supported overlays: ${nixpkgs.lib.concatStringsSep ", " (builtins.attrNames input.overlays)}"))
              inputAttrs.overlays or [ ];
          overlays = nixpkgs.lib.flatten (nixpkgs.lib.mapAttrsToList getOverlays (devenv.inputs or { }));
          pkgs = import nixpkgs {
            inherit system;
            config = {
              allowUnfree = devenv.allowUnfree or false;
              allowBroken = devenv.allowBroken or false;
              permittedInsecurePackages = devenv.permittedInsecurePackages or [ ];
            };
            inherit overlays;
          };
          lib = pkgs.lib;
          importModule = path:
            if lib.hasPrefix "./" path
            then if lib.hasSuffix ".nix" path
            then ./. + (builtins.substring 1 255 path)
            else ./. + (builtins.substring 1 255 path) + "/devenv.nix"
            else if lib.hasPrefix "../" path
            then throw "devenv: ../ is not supported for imports"
            else
              let
                paths = lib.splitString "/" path;
                name = builtins.head paths;
                input = inputs.${name} or (throw "Unknown input ${name}");
                subpath = "/${lib.concatStringsSep "/" (builtins.tail paths)}";
                devenvpath = "${input}" + subpath;
                devenvdefaultpath = devenvpath + "/devenv.nix";
              in
              if lib.hasSuffix ".nix" devenvpath
              then devenvpath
              else if builtins.pathExists devenvdefaultpath
              then devenvdefaultpath
              else throw (devenvdefaultpath + " file does not exist for input ${name}.");
          project = pkgs.lib.evalModules {
            specialArgs = inputs // { inherit inputs; };
            modules = [
              ({ config, ... }: {
                _module.args.pkgs = pkgs.appendOverlays (config.overlays or [ ]);
              })
              (inputs.devenv.modules + /top-level.nix)
              {
                devenv.cliVersion = version;
                devenv.root = devenv_root;
                devenv.dotfile = devenv_root + "/" + devenv_dotfile_string;
              }
              (pkgs.lib.optionalAttrs (inputs.devenv.isTmpDir or false) {
                devenv.tmpdir = devenv_tmpdir;
                devenv.runtime = devenv_runtime;
              })
              (pkgs.lib.optionalAttrs (inputs.devenv.hasIsTesting or false) {
                devenv.isTesting = devenv_istesting;
              })
              (pkgs.lib.optionalAttrs (container_name != null) {
                container.isBuilding = pkgs.lib.mkForce true;
                containers.${container_name}.isBuilding = true;
              })
              ({ options, ... }: {
                config.devenv = pkgs.lib.optionalAttrs (builtins.hasAttr "direnvrcLatestVersion" options.devenv) {
                  direnvrcLatestVersion = devenv_direnvrc_latest_version;
                };
              })
            ] ++ (map importModule (devenv.imports or [ ])) ++ [
              (if builtins.pathExists ./devenv.nix then ./devenv.nix else { })
              (devenv.devenv or { })
              (if builtins.pathExists ./devenv.local.nix then ./devenv.local.nix else { })
              (if builtins.pathExists (devenv_dotfile + "/cli-options.nix") then import (devenv_dotfile + "/cli-options.nix") else { })
            ];
          };
          config = project.config;

          options = pkgs.nixosOptionsDoc {
            options = builtins.removeAttrs project.options [ "_module" ];
            warningsAreErrors = false;
            # Unpack Nix types, e.g. literalExpression, mDoc.
            transformOptions =
              let isDocType = v: builtins.elem v [ "literalDocBook" "literalExpression" "literalMD" "mdDoc" ];
              in lib.attrsets.mapAttrs (_: v:
                if v ? _type && isDocType v._type then
                  v.text
                else if v ? _type && v._type == "derivation" then
                  v.name
                else
                  v
              );
          };

          build = options: config:
            lib.concatMapAttrs
              (name: option:
                if builtins.hasAttr "type" option then
                  if option.type.name == "output" || option.type.name == "outputOf" then {
                    ${name} = config.${name};
                  } else { }
                else
                  let v = build option config.${name};
                  in if v != { } then {
                    ${name} = v;
                  } else { }
              )
              options;

          systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
        in
        {
          devShell = lib.genAttrs systems (system: config.shell);
          packages = lib.genAttrs systems (system: {
            optionsJSON = options.optionsJSON;
            # deprecated
            inherit (config) info procfileScript procfileEnv procfile;
            ci = config.ciDerivation;
          });
          devenv = config;
          build = build project.options project.config;
        };
      }

# Try Argo using Nix
Nix is a package manager / build tool which focuses on reproducible build environments.
Argo Workflows has some basic support for Nix which is enough to get Argo Workflows up and running with minimal effort. 
Here are the steps to follow: 
  1. Modify the hosts file according to [this](https://argoproj.github.io/argo-workflows/running-locally/), don't worry about the other instructions. 
  2. Set up a k8s cluster, k3d is the recommended solution here. 
  3. Install [Nix](https://nixos.org/download.html).
  4. Run "nix develop --extra-experimental-features nix-command --extra-experimental-features flakes ./nix-files/ --impure" (you can add the extra features as a default in your nix.conf file).
  5. Run "devenv up". 

## Warning 
This is still barebones at the moment, any feature in the Makefile not mentioned here is excluded for now. 
In practice this means that only a "make start UI=true" equivalent is supported at the moment, as an additional caveat, there are no LDFlags set in the build, 
as a result the UI will show "0.0.0-unknown" for the version. 

## How do I upgrade a dependency? 
Most dependencies are in the Nix pkgs repository but if you want a specific version, you might have to build it yourself. 
This is fairly trivial in Nix, the idea is to just change the version string to whatever package you are concerned about. 

### Changing a python dependency version
If we look at the `mkdocs` dependency, we see a call to `buildPythonPackage`, to change the version we need to just modify the version string. 
Doing this will display a failure because the hash from the `fetchPypi` command will now differ, it will also display the correct hash, copy this hash 
and replace the existing hash value. 

### Changing a go dependency version 
The almost exact same principles apply here, the only difference being you must change the `vendorHash` and the `sha256` fields. 
The `vendorHash` is a hash of the vendored dependencies while the `sha256` is for the sources fetched from the `fetchFromGithub` call. 


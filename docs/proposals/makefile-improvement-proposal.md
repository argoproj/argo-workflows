# Proposal for Makefile improvements

## Introduction

The motivation for this proposal is to enable developers working on Argo Workflows to use build tools in a more reproducible way.
Currently the Makefile is unfortunately too opinionated and as a result is often a blocker when first setting up Argo Workflows locally.
I believe we should shrink the responsibilities of the Makefile and where possible outsource areas of responsibility to more specialized technology, such
as Devenv/Nix in the case of dependency management.

## Proposal Specifics

In order to better address reproducibility, it is better to split up the duties the Makefile currently performs into various sub components, that can be assembled in more appropriate technology. One important aspect here is to completely shift the responsibility of dependency management away from the Makefile and into technology such as Nix or Devenv. This proposal will also enable quicker access to a development build of Argo Workflows to developers, reducing the costs of on-boarding and barrier to entry.

### Devenv

#### Benefits of Devenv

- Reproducible build environment
- Ability to run processes

#### Disadvantages of Devenv

- Huge learning curve to tap into Nix functionality
- Less documentation

### Nix

#### Benefits of Nix

- Reproducible build environment
- Direct raw control of various Nix related functionality instead of using Devenv
- More documentation

#### Disadvantages of Nix

- Huge learning curve

### Recommendation

I suggest that we use Nix over Devenv. I believe that our build environment is unique enough that we will be tapping into Nix anyway, it probably makes sense to directly use Nix in that case.

### Proposal

In order to maximize the benefit we receive from using something like Nix, I suggest that we initially start off with a modest change to the Makefile.
The first proposal would be to remove out all dependency management code and replace this functionality with Nix, where it is trivially possible. This may not be possible for some go lang related binaries we use, we will retain the Makefile functionality in those cases, at least for a while. Eventually we will migrate more and more of this responsibility away from the Makefile. Following Nix being responsible for all dependency management, we could start to consider moving more of our build system itself into Nix, perhaps it is easiest to start off with UI build as it is relatively painless. However, do note that this is not a requirement, I do not see a problem with the Makefile and the Nix file co-existing, it is more about finding a good balance between the reproducibility we desire and the effort we put into obtaining said reproducibility. An example for a replacement could be [this dependency](https://github.com/argoproj/argo-workflows/blob/047952afd539d06cae2fd6ba0b608b19c1194bba/Makefile#L626) for example, note that we do not state any version here, replacing such installations with Nix based installations will ensure that we can ensure that if a build works on a certain developer's machine, it should also work on every other machine as well.

### What will Nix get us?

As mentioned previously Nix gets us closer to reproducible build environments. It should ease significantly the on-boarding process of developers onto the project.
There have been several developers who wanted to work on Argo Workflows but found the Makefile to be a barrier, it is likely that there are more developers on this boat. With a reproducible build environment, we hope that
everyone who would like to contribute to the project is able to do so easily. It should also save time for engineers on-boarding onto the project, especially if they are using a system that is not Ubuntu or OSX.

### What will Nix cost us?

If we proceed further with Nix, it will require some amount of people working on Argo Workflows to learn it, this is not a trivial task by any means.
It will increase the barrier when it comes to changes that are build related, however, this isn't necessarily bad as build related changes should be far less frequent, the friction we will endure here is likely manageable.

### How will developers use nix?

In the case that both Nix and the Makefile co-exist, we could use nix inside the Makefile itself. The Makefile calls into Nix to setup a developer environment with all dependencies, it will then continue the rest of the Makefile execution as normal.
Following a complete or near complete migration to Nix, we can use `nix-build` for more of our tasks. An example of a C++ project environment is provided [here](https://blog.galowicz.de/2019/04/17/tutorial_nix_cpp_setup/)

## Resources

- [Nix Manual - Go](https://nixos.org/manual/nixpkgs/stable/#sec-language-go)
- [Devenv](https://devenv.sh/)
- [How to Learn Nix](https://ianthehenry.com/posts/how-to-learn-nix/)

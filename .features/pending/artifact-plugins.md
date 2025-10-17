Description: Artifact Drivers as plugins
Author: [Alan Clucas](https://github.com/Joibel), [JP Zivalich](https://github.com/JPZ13), [Elliot Gunton](https://github.com/elliotgunton)
Component: General
Issues: 5862

Artifact Drivers can now be added via a plugin mechanism.
You can write a GRPC server which acts as an artifact driver to upload and download artifacts to a repository, and supply that as a container image.
Argo workflows can then use that as a driver.

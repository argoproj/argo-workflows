Description: Migrate protobuf codegen from gogo/protobuf to protoc-gen-go and grpc-gateway v2
Authors: [Alan Clucas](https://github.com/Joibel)
Component: Build and Development
Issues: 7400 16595

The API client and server stubs under `pkg/apiclient` are now generated with the officially maintained `protoc-gen-go` and `protoc-gen-go-grpc` instead of the unmaintained gogo/protobuf fork.
The HTTP gateway moved from grpc-gateway v1 to v2, and `protoc-gen-openapiv2` replaces `protoc-gen-swagger`.

This is largely internal, but has some API-visible effects:

    - The `/api/v1/stream/events/{namespace}` stream now wraps each event as `{"result": {"type": ..., "object": ...}}` instead of `{"result": <Event>}`, matching the other watch streams.
    - Some OpenAPI definition names changed (for example `WorkflowCreateRequest` is now `CreateWorkflowBody`), which renames the corresponding generated SDK classes.
    - HTTP error bodies now use the standard `google.rpc.Status` shape instead of grpc-gateway v1's error types.

See the upgrading guide for details.

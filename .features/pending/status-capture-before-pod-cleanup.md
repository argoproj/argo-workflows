Description: Preserve captured Pod results before removing the status finalizer
Authors: [Ruslan Shaydullin](https://github.com/ruslan-shaydullin)
Component: General
Issues: 17024

With `ARGO_POD_STATUS_CAPTURE_FINALIZER=true`, Pod cleanup verifies the saved node result and Pod UID instead of relying on the age of a deleting Pod.
The optional `capturedPodUID` node status field allows the controller to recover cleanup after restarting, including for completed Workflows.
Pods of Workflows that completed under an older controller, whose saved results cannot be matched to a Pod UID, keep the status finalizer, with the reason in the controller log, until you act.
Installations using full CRDs must update them with the controller.
See [Pod Status Capture and Retained Pods](status-capture.md) for upgrade and cleanup behavior.

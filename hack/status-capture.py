#!/usr/bin/env python3
"""Inspect saved Argo Pod capture evidence and prepare an explicit operator release.

This program never calls Kubernetes, SQL, or a network service. Its inventory is
an observation of supplied files, not an authoritative cleanup permission.
"""

import argparse
import base64
import copy
import datetime
import gzip
import hashlib
import json
import os
from pathlib import Path
import sys

DOMAIN = "workflows.argoproj.io/"
FINALIZER = DOMAIN + "status"
TERMINAL = {"Succeeded", "Failed", "Error"}


class EvidenceError(ValueError):
    pass


def read_json(path):
    with open(path, encoding="utf-8") as stream:
        return json.load(stream)


def objects(value):
    if isinstance(value, list):
        result = value
    elif isinstance(value, dict) and isinstance(value.get("items"), list):
        result = value["items"]
    elif isinstance(value, dict):
        result = [value]
    else:
        raise EvidenceError("expected a JSON object or list")
    if not all(isinstance(item, dict) for item in result):
        raise EvidenceError("list contains a non-object")
    return result


def single(path, kind):
    values = objects(read_json(path))
    if len(values) != 1 or values[0].get("kind") != kind:
        raise EvidenceError("expected exactly one " + kind)
    return values[0]


def metadata(obj):
    return obj.get("metadata") or {}


def object_key(obj):
    meta = metadata(obj)
    return meta.get("namespace"), meta.get("name")


def unique_index(values):
    indexed = {}
    for value in values:
        key = object_key(value)
        if key in indexed:
            raise EvidenceError("duplicate namespace/name in input")
        indexed[key] = value
    return indexed


def require_identity(obj):
    meta = metadata(obj)
    if not all(isinstance(meta.get(key), str) and meta[key]
               for key in ("namespace", "name", "uid", "resourceVersion")):
        raise EvidenceError("namespace/name/UID/resourceVersion must all be present")


def require_owner(pod, wf):
    require_identity(pod)
    require_identity(wf)
    pm, wm = metadata(pod), metadata(wf)
    owners = [owner for owner in pm.get("ownerReferences", [])
              if owner.get("controller") is True]
    if len(owners) != 1:
        raise EvidenceError("exactly one controller owner is required")
    owner = owners[0]
    if (owner.get("apiVersion") != "argoproj.io/v1alpha1"
            or owner.get("kind") != "Workflow"
            or owner.get("name") != wm["name"]
            or owner.get("uid") != wm["uid"]
            or pm["namespace"] != wm["namespace"]):
        raise EvidenceError("owner identity does not match Workflow namespace/name/UID")
    labels, wf_labels = pm.get("labels", {}), wm.get("labels", {})
    if labels.get(DOMAIN + "workflow", wm["name"]) != wm["name"]:
        raise EvidenceError("workflow label contradicts owner")
    if labels.get(DOMAIN + "controller-instanceid", "") != wf_labels.get(DOMAIN + "controller-instanceid", ""):
        raise EvidenceError("controller instance identity differs")
    if labels.get(DOMAIN + "component"):
        raise EvidenceError("component Pods use a different result channel")


def hydrated_nodes(wf, offloads, hydrated):
    status = wf.get("status", {})
    raw, compressed = status.get("nodes"), status.get("compressedNodes")
    version = status.get("offloadNodeStatusVersion")
    if sum(bool(value) for value in (raw, compressed, version)) > 1:
        raise EvidenceError("contradictory node storage representations")
    if raw:
        if not isinstance(raw, dict):
            raise EvidenceError("nodes must be an object")
        return raw, "raw"
    candidate = hydrated.get(object_key(wf))
    if candidate is not None:
        before, after = metadata(wf), metadata(candidate)
        if any(before.get(key) != after.get(key) for key in ("namespace", "name", "uid", "resourceVersion")):
            raise EvidenceError("hydrated Workflow identity/resourceVersion differs")
        result = candidate.get("status", {})
        if result.get("compressedNodes") or result.get("offloadNodeStatusVersion") or not isinstance(result.get("nodes"), dict):
            raise EvidenceError("supplied server response is not hydrated")
        original_status, result_status = copy.deepcopy(status), copy.deepcopy(result)
        for key in ("nodes", "compressedNodes", "offloadNodeStatusVersion"):
            original_status.pop(key, None)
            result_status.pop(key, None)
        if original_status != result_status or wf.get("spec") != candidate.get("spec"):
            raise EvidenceError("hydrated Workflow semantic metadata differs")
        return result["nodes"], "server-hydrated"
    if version:
        wm = metadata(wf)
        matches = [row for row in offloads if row.get("namespace") == wm.get("namespace")
                   and row.get("uid") == wm.get("uid") and row.get("version") == version]
        if len(matches) != 1 or not isinstance(matches[0].get("nodes"), dict):
            raise EvidenceError("exact referenced offload snapshot is unavailable or ambiguous")
        return matches[0]["nodes"], "offloaded"
    if compressed:
        try:
            packed = base64.b64decode(compressed, validate=True)
            if not packed.startswith(b"\x1f\x8b"):
                raise EvidenceError("non-gzip compression requires a matching hydrated server response")
            result = json.loads(gzip.decompress(packed))
            if not isinstance(result, dict):
                raise EvidenceError("decompressed nodes must be an object")
            return result, "compressed-gzip"
        except (ValueError, OSError, EOFError) as error:
            raise EvidenceError("cannot decode node data: " + str(error)) from error
    if isinstance(raw, dict):
        return raw, "raw"
    raise EvidenceError("node data is unavailable")


def matching_node(pod, nodes):
    annotations = metadata(pod).get("annotations", {})
    node_id = annotations.get(DOMAIN + "node-id")
    node_name = annotations.get(DOMAIN + "node-name")
    if not node_id:
        matches = [key for key, value in nodes.items() if node_name and value.get("name") == node_name]
        if len(matches) != 1:
            raise EvidenceError("node identity is unavailable or ambiguous")
        node_id = matches[0]
    node = nodes.get(node_id)
    if not isinstance(node, dict) or node.get("id") != node_id or not node.get("name"):
        raise EvidenceError("persisted node identity is unavailable")
    if node_name and node.get("name") != node_name:
        raise EvidenceError("Pod node name differs from persisted node")
    if node.get("type") != "Pod":
        raise EvidenceError("node is not an ordinary Pod node")
    return node_id, node


def task_state(wf, node_id, node):
    node_state = node.get("taskResultSynced")
    legacy = wf.get("status", {}).get("taskResultsCompletionStatus")
    legacy_state = legacy.get(node_id) if isinstance(legacy, dict) else None
    if node_state is False or legacy_state is False:
        return "pending"
    if node_state is True or legacy_state is True:
        return "explicit-complete"
    return "legacy-unspecified"


def legacy_class(wf, node, nodes):
    """Triage hints only: this intentionally does not duplicate the verifier."""
    spec = wf.get("spec", {})
    templates = spec.get("templates", []) + wf.get("status", {}).get("storedWorkflowSpec", {}).get("templates", [])
    templates += list(wf.get("status", {}).get("storedTemplates", {}).values())
    template = next((item for item in templates if item.get("name") == node.get("templateName")), {})
    if node.get("memoizationStatus") or template.get("memoize"):
        return "memoization-transformation"
    if node.get("daemoned") or template.get("daemon") or spec.get("shutdown"):
        return "daemon-or-shutdown"
    if node.get("failedPodRestarts") or node.get("restartingPodUID") or node.get("nodeFlag"):
        return "restart-or-execution-history"
    if node.get("phase") != "Succeeded":
        return "controller-error-or-non-success"
    outputs = dict(node.get("outputs") or {})
    outputs.pop("exitCode", None)
    if outputs or template.get("outputs"):
        return "external-outputs"
    if len(nodes) != 1 or node.get("name") != metadata(wf).get("name"):
        return "multiple-nodes"
    if not template.get("container"):
        return "template-unavailable-or-not-container"
    return "possible-simple-success-controller-verification-required"


def inspect(pod, wf, offloads, hydrated):
    pm = metadata(pod)
    report = {"namespace": pm.get("namespace"), "pod": pm.get("name"),
              "podUID": pm.get("uid"), "podResourceVersion": pm.get("resourceVersion"),
              "reason": "identity-conflict", "automaticCaptureProved": False}
    if FINALIZER not in pm.get("finalizers", []):
        report["reason"] = "no-status-finalizer"
        return report, None
    if wf is None:
        report["reason"] = "owner-not-in-snapshot"
        return report, None
    wm = metadata(wf)
    report.update(workflow=wm.get("name"), workflowUID=wm.get("uid"),
                  workflowResourceVersion=wm.get("resourceVersion"))
    try:
        require_owner(pod, wf)
    except EvidenceError as error:
        report["detail"] = str(error)
        return report, None
    if wm.get("deletionTimestamp"):
        report["reason"] = "owner-deleting"
        return report, None
    try:
        nodes, storage = hydrated_nodes(wf, offloads, hydrated)
    except EvidenceError as error:
        report.update(reason="data-unavailable", detail=str(error))
        return report, None
    report["storage"] = storage
    try:
        node_id, node = matching_node(pod, nodes)
    except EvidenceError as error:
        report["detail"] = str(error)
        return report, None
    report.update(nodeID=node_id, nodePhase=node.get("phase"), capturedPodUID=node.get("capturedPodUID", ""),
                  taskResult=task_state(wf, node_id, node))
    state = {"nodes": nodes, "node": node, "nodeID": node_id}
    pod_phase, node_phase = pod.get("status", {}).get("phase"), node.get("phase")
    if pod_phase not in {"Succeeded", "Failed"}:
        report["reason"] = "pod-not-terminal"
    elif node_phase == "Pending" and node.get("restartingPodUID") == pm.get("uid"):
        report["reason"] = "persisted-restart-disposition"
    elif node.get("capturedPodUID") and node["capturedPodUID"] != pm.get("uid"):
        report["reason"] = "capture-identity-conflict"
    elif report["taskResult"] == "pending" or node_phase not in TERMINAL:
        report["reason"] = "result-or-task-pending"
    elif node.get("capturedPodUID") == pm.get("uid"):
        report["reason"] = "matching-receipt-in-snapshot"
    elif wm.get("labels", {}).get(DOMAIN + "completed") != "true" or wf.get("status", {}).get("phase") not in TERMINAL:
        report["reason"] = "active-workflow-awaiting-capture"
    else:
        report["reason"] = "legacy-no-receipt"
        report["legacyClass"] = legacy_class(wf, node, nodes)
        report["detail"] = "controller must prove supported same-result capture; otherwise retain or explicitly dispose"
    return report, state


def private_json(path, value):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write("\n")


def dependencies(args):
    offloads = []
    for path in args.offload:
        offloads.extend(objects(read_json(path)))
    hydrated = unique_index(objects(read_json(args.hydrated_workflows))) if args.hydrated_workflows else {}
    return offloads, hydrated


def one_report(args):
    pod, wf = single(args.pod, "Pod"), single(args.workflow, "Workflow")
    offloads, hydrated = dependencies(args)
    report, state = inspect(pod, wf, offloads, hydrated)
    if state is None:
        raise EvidenceError(report["reason"] + ": " + report.get("detail", "exact evidence unavailable"))
    return pod, wf, report, state


def prepare_release(args):
    if not args.acknowledge_unproven_capture:
        raise EvidenceError("explicit --acknowledge-unproven-capture is required; export is not capture proof")
    pod, _, report, _ = one_report(args)
    if report["reason"] != "legacy-no-receipt":
        raise EvidenceError("release requires a completed legacy result with no receipt, not " + report["reason"])
    finalizers = metadata(pod).get("finalizers", [])
    if finalizers.count(FINALIZER) != 1:
        raise EvidenceError("exactly one status finalizer is required")
    patch = [
        {"op": "test", "path": "/metadata/uid", "value": metadata(pod)["uid"]},
        {"op": "test", "path": "/metadata/resourceVersion", "value": metadata(pod)["resourceVersion"]},
        {"op": "test", "path": "/metadata/finalizers", "value": finalizers},
        {"op": "remove", "path": "/metadata/finalizers/" + str(finalizers.index(FINALIZER))},
    ]
    private_json(args.output, patch)
    return {"patch": str(args.output), "evidence": report,
            "decision": "explicit operator disposition; automatic Pod capture is not proved",
            "applied": False}


def export_evidence(args):
    pod, wf, report, state = one_report(args)
    destination = Path(args.output_dir)
    destination.mkdir(mode=0o700)
    files = {"pod.json": pod, "workflow.json": wf, "nodes.json": state["nodes"]}
    if args.hydrated_workflows:
        files["server-response.json"] = read_json(args.hydrated_workflows)
    for number, path in enumerate(args.offload):
        files["offload-" + str(number) + ".json"] = read_json(path)
    if args.task_results:
        files["task-results.json"] = read_json(args.task_results)
    manifest = {"createdAt": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                "observation": report, "automaticCaptureProved": False,
                "purpose": "private evidence, not a Workflow capture receipt", "sha256": {}}
    for name, value in files.items():
        path = destination / name
        private_json(path, value)
        manifest["sha256"][name] = hashlib.sha256(path.read_bytes()).hexdigest()
    private_json(destination / "manifest.json", manifest)
    return {"directory": str(destination), "evidence": report, "files": list(files)}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    inventory = commands.add_parser("inventory", help="classify supplied Pod/Workflow snapshots")
    inventory.add_argument("--pods", required=True)
    inventory.add_argument("--workflows", required=True)
    export = commands.add_parser("export", help="validate and save private evidence, without asserting capture")
    export.add_argument("--output-dir", required=True)
    export.add_argument("--task-results")
    release = commands.add_parser("prepare-release", help="prepare, never apply, an explicit operator patch")
    release.add_argument("--acknowledge-unproven-capture", action="store_true")
    release.add_argument("--output", required=True)
    for command in (inventory, export, release):
        command.add_argument("--offload", action="append", default=[], help="exact {namespace,uid,version,nodes} SQL snapshot")
        command.add_argument("--hydrated-workflows", help="authoritative hydrated server response at the same UID/resourceVersion")
        if command is not inventory:
            command.add_argument("--pod", required=True)
            command.add_argument("--workflow", required=True)
    args = parser.parse_args(argv)
    try:
        if args.command == "inventory":
            workflows = unique_index(objects(read_json(args.workflows)))
            offloads, hydrated = dependencies(args)
            rows = []
            for pod in objects(read_json(args.pods)):
                if FINALIZER not in metadata(pod).get("finalizers", []):
                    continue
                owners = [owner for owner in metadata(pod).get("ownerReferences", []) if owner.get("controller") is True]
                name = owners[0].get("name") if len(owners) == 1 else None
                wf = workflows.get((metadata(pod).get("namespace"), name))
                rows.append(inspect(pod, wf, offloads, hydrated)[0])
            result = {"source": "offline snapshots; re-read the API before action", "pods": rows}
        elif args.command == "export":
            result = export_evidence(args)
        else:
            result = prepare_release(args)
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0
    except (EvidenceError, OSError, ValueError, TypeError, KeyError) as error:
        print(json.dumps({"error": str(error), "applied": False}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())

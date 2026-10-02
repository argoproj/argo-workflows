"""Offline tests for the status-capture operator helper (Python standard library only)."""

import argparse
import base64
import contextlib
import copy
import gzip
import importlib.util
import io
import json
import os
from pathlib import Path
import stat
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parents[1] / 'status-capture.py'
spec = importlib.util.spec_from_file_location('status_capture', SOURCE)
tool = importlib.util.module_from_spec(spec)
spec.loader.exec_module(tool)


def fixture():
    wf = {'apiVersion': 'argoproj.io/v1alpha1', 'kind': 'Workflow',
          'metadata': {'name': 'example', 'namespace': 'capture', 'uid': 'workflow-uid', 'resourceVersion': '10',
                       'labels': {tool.DOMAIN + 'completed': 'true'}},
          'spec': {'templates': [{'name': 'main', 'container': {'image': 'busybox'}}]},
          'status': {'phase': 'Error', 'message': 'memoization Save failed',
                     'nodes': {'example': {'id': 'example', 'name': 'example', 'type': 'Pod', 'templateName': 'main',
                                          'phase': 'Error', 'message': 'cache write forbidden', 'taskResultSynced': True,
                                          'memoizationStatus': {'key': 'x', 'cacheName': 'cache'}, 'outputs': {'exitCode': '0'}}}}}
    pod = {'apiVersion': 'v1', 'kind': 'Pod',
           'metadata': {'name': 'example-pod', 'namespace': 'capture', 'uid': 'pod-uid', 'resourceVersion': '20',
                        'finalizers': ['another.example/keep', tool.FINALIZER, 'another.example/last'],
                        'annotations': {tool.DOMAIN + 'node-id': 'example', tool.DOMAIN + 'node-name': 'example'},
                        'ownerReferences': [{'apiVersion': 'argoproj.io/v1alpha1', 'kind': 'Workflow', 'name': 'example',
                                             'uid': 'workflow-uid', 'controller': True}]},
           'status': {'phase': 'Succeeded'}}
    return pod, wf


def storage_error_fixture():
    pod, wf = fixture()
    wf['status']['message'] = ('workflow is longer than maximum allowed size. compressed size 4096 > maxSize 2048'
                               'Tried to offload but encountered error: offload node status is not supported')
    node = wf['status']['nodes']['example']
    node.update(phase='Running', taskResultSynced=False)
    wf['status']['taskResultsCompletionStatus'] = {'example': False}
    return pod, wf


class OperatorTests(unittest.TestCase):
    def setUp(self):
        self.pod, self.wf = fixture()

    def inspect(self):
        return tool.inspect(self.pod, self.wf, [], {})[0]

    def test_memoization_error_inventory_not_claimed_capture(self):
        row = self.inspect()
        self.assertEqual(row['reason'], 'legacy-no-receipt')
        self.assertEqual(row['legacyClass'], 'memoization-transformation')
        self.assertFalse(row['automaticCaptureProved'])

    def test_null_optional_template_sources_are_absent(self):
        for source in ('spec', 'storedWorkflowSpec', 'storedTemplates', None):
            with self.subTest(source=source):
                self.pod, self.wf = fixture()
                template = self.wf['spec']['templates'][0]
                self.wf['spec']['templates'] = None
                self.wf['status'].update(storedWorkflowSpec=None, storedTemplates=None, phase='Succeeded')
                node = self.wf['status']['nodes']['example']
                node.pop('memoizationStatus')
                node['phase'] = 'Succeeded'
                if source == 'spec':
                    self.wf['spec']['templates'] = [template]
                elif source == 'storedWorkflowSpec':
                    self.wf['status']['storedWorkflowSpec'] = {'templates': [template]}
                elif source == 'storedTemplates':
                    self.wf['status']['storedWorkflowSpec'] = {'templates': None}
                    self.wf['status']['storedTemplates'] = {'scope/main': template}
                else:
                    self.wf['spec'] = None
                row = self.inspect()
                expected = ('possible-simple-success-controller-verification-required' if source
                            else 'template-unavailable-or-not-container')
                self.assertEqual(row['legacyClass'], expected)
                self.assertEqual(row['reason'], 'legacy-no-receipt')
                self.assertFalse(row['automaticCaptureProved'])

    def test_unnamed_template_is_not_a_match_for_missing_node_template_name(self):
        node = self.wf['status']['nodes']['example']
        node.pop('memoizationStatus')
        node.pop('templateName')
        node['phase'] = 'Succeeded'
        self.wf['status']['phase'] = 'Succeeded'
        self.wf['spec']['templates'][0].pop('name')
        self.assertEqual(self.inspect()['legacyClass'], 'template-unavailable-or-not-container')

    def test_malformed_template_sources_produce_cli_error_without_artifacts(self):
        cases = ((('spec',), []),
                 (('spec', 'templates'), {}),
                 (('spec', 'templates'), [None]),
                 (('status', 'storedWorkflowSpec'), []),
                 (('status', 'storedWorkflowSpec', 'templates'), 'invalid'),
                 (('status', 'storedWorkflowSpec', 'templates'), [None]),
                 (('status', 'storedTemplates'), []),
                 (('status', 'storedTemplates'), {'scope/main': None}))
        for path, value in cases:
            with self.subTest(path=path, value=value), tempfile.TemporaryDirectory() as directory:
                self.pod, self.wf = fixture()
                target = self.wf
                for key in path[:-1]:
                    target = target.setdefault(key, {})
                target[path[-1]] = value
                args = self.files(directory)
                commands = (['inventory', '--pods', args.pod, '--workflows', args.workflow],
                            ['export', '--pod', args.pod, '--workflow', args.workflow,
                             '--output-dir', args.output_dir],
                            ['prepare-release', '--pod', args.pod, '--workflow', args.workflow,
                             '--output', args.output, '--acknowledge-unproven-capture'])
                for command in commands:
                    with self.subTest(command=command[0]):
                        output, errors = io.StringIO(), io.StringIO()
                        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(errors):
                            code = tool.main(command)
                        self.assertEqual(code, 2)
                        self.assertEqual(output.getvalue(), '')
                        error = json.loads(errors.getvalue())
                        self.assertIn('.'.join(path), error['error'])
                        self.assertFalse(error['applied'])
                        self.assertFalse(Path(args.output).exists())
                        self.assertFalse(Path(args.output_dir).exists())

    def test_missing_owner_snapshot_is_not_orphan(self):
        row, _ = tool.inspect(self.pod, None, [], {})
        self.assertEqual(row['reason'], 'owner-not-in-snapshot')

    def test_wrong_owner_namespace_instance_and_node_fail(self):
        for mutate in (lambda p, w: p['metadata']['ownerReferences'][0].update(uid='foreign'),
                       lambda p, w: w['metadata'].update(namespace='other'),
                       lambda p, w: p['metadata'].update(labels={tool.DOMAIN + 'controller-instanceid': 'other'}),
                       lambda p, w: p['metadata']['annotations'].update({tool.DOMAIN + 'node-name': 'other'})):
            with self.subTest(mutate=mutate):
                pod, wf = fixture()
                mutate(pod, wf)
                self.assertEqual(tool.inspect(pod, wf, [], {})[0]['reason'], 'identity-conflict')

    def test_task_result_contradiction_fails(self):
        self.wf['status']['taskResultsCompletionStatus'] = {'example': False}
        self.assertEqual(self.inspect()['reason'], 'result-or-task-pending')

    def test_matching_and_conflicting_receipt(self):
        node = self.wf['status']['nodes']['example']
        node['capturedPodUID'] = 'pod-uid'
        self.assertEqual(self.inspect()['reason'], 'matching-receipt-in-snapshot')
        node['capturedPodUID'] = 'other'
        self.assertEqual(self.inspect()['reason'], 'capture-identity-conflict')

    def test_compressed_and_offload_exact_reference(self):
        nodes = self.wf['status'].pop('nodes')
        self.wf['status']['compressedNodes'] = base64.b64encode(gzip.compress(json.dumps(nodes).encode())).decode()
        self.assertEqual(tool.hydrated_nodes(self.wf, [], {})[0], nodes)
        self.wf['status'].pop('compressedNodes')
        self.wf['status']['offloadNodeStatusVersion'] = 'fnv:123'
        row = {'namespace': 'capture', 'uid': 'workflow-uid', 'version': 'fnv:123', 'nodes': nodes}
        self.assertEqual(tool.hydrated_nodes(self.wf, [row], {})[0], nodes)
        for key in ('uid', 'namespace', 'version'):
            wrong = dict(row, **{key: 'wrong'})
            with self.subTest(key=key), self.assertRaises(tool.EvidenceError):
                tool.hydrated_nodes(self.wf, [wrong], {})
        with self.assertRaises(tool.EvidenceError):
            tool.hydrated_nodes(self.wf, [row, row], {})

    def test_server_hydration_must_match_uid_rv_and_semantics(self):
        server = copy.deepcopy(self.wf)
        self.wf['status'].pop('nodes')
        self.wf['status']['offloadNodeStatusVersion'] = 'fnv:123'
        index = {('capture', 'example'): server}
        self.assertEqual(tool.hydrated_nodes(self.wf, [], index)[0], server['status']['nodes'])
        for key in ('uid', 'resourceVersion'):
            old = server['metadata'][key]
            server['metadata'][key] = 'other'
            with self.subTest(key=key), self.assertRaises(tool.EvidenceError):
                tool.hydrated_nodes(self.wf, [], index)
            server['metadata'][key] = old
        server['status']['message'] = 'changed'
        with self.assertRaises(tool.EvidenceError):
            tool.hydrated_nodes(self.wf, [], index)

    def test_non_gzip_is_explicit_hold_without_server(self):
        self.wf['status'].pop('nodes')
        self.wf['status']['compressedNodes'] = base64.b64encode(b'\x28\xb5\x2f\xfd').decode()
        self.assertEqual(self.inspect()['reason'], 'data-unavailable')

    def files(self, directory):
        pod_path, wf_path = Path(directory) / 'p.json', Path(directory) / 'w.json'
        pod_path.write_text(json.dumps(self.pod))
        wf_path.write_text(json.dumps(self.wf))
        return argparse.Namespace(pod=str(pod_path), workflow=str(wf_path), offload=[], hydrated_workflows=None,
                                  acknowledge_unproven_capture=True, acknowledge_incomplete_result=False,
                                  output=str(Path(directory) / 'patch.json'),
                                  output_dir=str(Path(directory) / 'export'), task_results=None)

    def test_storage_error_inventory_and_export_preserve_incomplete_result(self):
        self.pod, self.wf = storage_error_fixture()
        before = copy.deepcopy(self.wf)
        with tempfile.TemporaryDirectory() as directory:
            args = self.files(directory)
            result = tool.export_evidence(args)
            row = result['evidence']
            self.assertEqual(row['reason'], 'completed-storage-error-without-capture')
            self.assertEqual(row['taskResult'], 'pending')
            self.assertEqual(row['nodePhase'], 'Running')
            self.assertFalse(row['automaticCaptureProved'])
            self.assertEqual(json.loads((Path(args.output_dir) / 'workflow.json').read_text()), before)
            self.assertEqual(json.loads((Path(args.output_dir) / 'nodes.json').read_text()), before['status']['nodes'])
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                code = tool.main(['inventory', '--pods', args.pod, '--workflows', args.workflow])
            self.assertEqual(code, 0)
            self.assertEqual(json.loads(output.getvalue())['pods'][0], row)
        self.assertEqual(self.wf, before)

    def test_storage_error_release_requires_both_acknowledgements(self):
        for capture_ack, incomplete_ack in ((False, False), (False, True), (True, False), (True, True)):
            with self.subTest(capture_ack=capture_ack, incomplete_ack=incomplete_ack), tempfile.TemporaryDirectory() as directory:
                self.pod, self.wf = storage_error_fixture()
                before_pod, before_wf = copy.deepcopy(self.pod), copy.deepcopy(self.wf)
                args = self.files(directory)
                command = ['prepare-release', '--pod', args.pod, '--workflow', args.workflow, '--output', args.output]
                if capture_ack:
                    command.append('--acknowledge-unproven-capture')
                if incomplete_ack:
                    command.append('--acknowledge-incomplete-result')
                output, errors = io.StringIO(), io.StringIO()
                with contextlib.redirect_stdout(output), contextlib.redirect_stderr(errors):
                    code = tool.main(command)
                if capture_ack and incomplete_ack:
                    self.assertEqual(code, 0)
                    result = json.loads(output.getvalue())
                    self.assertFalse(result['applied'])
                    self.assertFalse(result['evidence']['automaticCaptureProved'])
                    patch = json.loads(Path(args.output).read_text())
                    self.assertEqual(patch, [
                        {'op': 'test', 'path': '/metadata/uid', 'value': before_pod['metadata']['uid']},
                        {'op': 'test', 'path': '/metadata/resourceVersion', 'value': before_pod['metadata']['resourceVersion']},
                        {'op': 'test', 'path': '/metadata/finalizers', 'value': before_pod['metadata']['finalizers']},
                        {'op': 'remove', 'path': '/metadata/finalizers/1'},
                    ])
                else:
                    self.assertEqual(code, 2)
                    self.assertFalse(json.loads(errors.getvalue())['applied'])
                    self.assertFalse(Path(args.output).exists())
                self.assertEqual(self.pod, before_pod)
                self.assertEqual(self.wf, before_wf)

    def test_storage_error_acknowledgement_does_not_waive_other_holds(self):
        cases = (lambda p, w: w['metadata']['labels'].update({tool.DOMAIN + 'completed': 'false'}),
                 lambda p, w: w['status'].update(phase='Running'),
                 lambda p, w: w['status'].update(message='offload node status is not supported'),
                 lambda p, w: w['status'].update(message='workflow is longer than maximum allowed size. SQL unavailable'),
                 lambda p, w: w['status'].update(message='ordinary workflow error'),
                 lambda p, w: p['status'].update(phase='Running'),
                 lambda p, w: w['status']['nodes']['example'].update(capturedPodUID='other'),
                 lambda p, w: w['status']['nodes']['example'].update(capturedPodUID='pod-uid'),
                 lambda p, w: w['status']['nodes']['example'].update(restartingPodUID='pod-uid', phase='Pending'),
                 lambda p, w: (w['status']['nodes']['example'].update(restartingPodUID='other', phase='Succeeded', taskResultSynced=True),
                               w['status']['taskResultsCompletionStatus'].update(example=True)),
                 lambda p, w: p['metadata']['ownerReferences'][0].update(uid='foreign'),
                 lambda p, w: w['status'].update(nodes={}))
        for number, mutate in enumerate(cases):
            with self.subTest(case=number), tempfile.TemporaryDirectory() as directory:
                self.pod, self.wf = storage_error_fixture()
                mutate(self.pod, self.wf)
                args = self.files(directory)
                args.acknowledge_incomplete_result = True
                with self.assertRaises(tool.EvidenceError):
                    tool.prepare_release(args)
                self.assertFalse(Path(args.output).exists())

    def test_release_only_own_finalizer_with_uid_rv_preconditions(self):
        before_pod, before_wf = copy.deepcopy(self.pod), copy.deepcopy(self.wf)
        with tempfile.TemporaryDirectory() as directory:
            args = self.files(directory)
            result = tool.prepare_release(args)
            patch = json.loads(Path(args.output).read_text())
            self.assertFalse(result['applied'])
            self.assertEqual([op['path'] for op in patch], ['/metadata/uid', '/metadata/resourceVersion',
                                                         '/metadata/finalizers', '/metadata/finalizers/1'])
            self.assertEqual([op['op'] for op in patch], ['test', 'test', 'test', 'remove'])
            self.assertEqual(patch[0]['value'], before_pod['metadata']['uid'])
            self.assertEqual(patch[1]['value'], before_pod['metadata']['resourceVersion'])
            self.assertEqual(patch[2]['value'], before_pod['metadata']['finalizers'])
            self.assertEqual(stat.S_IMODE(os.stat(args.output).st_mode), 0o600)
        self.assertEqual(self.pod, before_pod)
        self.assertEqual(self.wf, before_wf)

    def test_release_requires_ack_and_rejects_pending_or_different_uid(self):
        with tempfile.TemporaryDirectory() as directory:
            args = self.files(directory)
            args.acknowledge_unproven_capture = False
            with self.assertRaises(tool.EvidenceError):
                tool.prepare_release(args)
        for node_update in ({'taskResultSynced': False}, {'capturedPodUID': 'other'}, {'capturedPodUID': 'pod-uid'}):
            self.pod, self.wf = fixture()
            self.wf['status']['nodes']['example'].update(node_update)
            with self.subTest(node_update=node_update), tempfile.TemporaryDirectory() as directory:
                with self.assertRaises(tool.EvidenceError):
                    tool.prepare_release(self.files(directory))

    def test_release_rejects_active_owner_and_duplicate_finalizer(self):
        for mutate in (lambda: self.wf['metadata']['labels'].update({tool.DOMAIN + 'completed': 'false'}),
                       lambda: self.pod['metadata']['finalizers'].append(tool.FINALIZER)):
            self.pod, self.wf = fixture()
            mutate()
            with tempfile.TemporaryDirectory() as directory, self.assertRaises(tool.EvidenceError):
                tool.prepare_release(self.files(directory))

    def test_private_export_validates_and_preserves_semantics(self):
        with tempfile.TemporaryDirectory() as directory:
            args = self.files(directory)
            tool.export_evidence(args)
            exported = Path(args.output_dir)
            self.assertEqual(stat.S_IMODE(exported.stat().st_mode), 0o700)
            for path in exported.iterdir():
                self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o600)
            self.assertEqual(json.loads((exported / 'workflow.json').read_text()), self.wf)
            exported_nodes = json.loads((exported / 'nodes.json').read_text())
            self.assertNotIn('capturedPodUID', exported_nodes['example'])
            self.assertNotIn('capturedPodUID', self.wf['status']['nodes']['example'])
            manifest = json.loads((exported / 'manifest.json').read_text())
            self.assertFalse(manifest['automaticCaptureProved'])
            self.assertEqual(len(manifest['sha256']), 3)
            with self.assertRaises(FileExistsError):
                tool.export_evidence(args)

    def test_unavailable_evidence_does_not_create_export_or_release(self):
        for storage in ({}, {'offloadNodeStatusVersion': 'missing-version'}):
            with self.subTest(storage=storage), tempfile.TemporaryDirectory() as directory:
                self.pod, self.wf = fixture()
                self.wf['status'].pop('nodes')
                self.wf['status'].update(storage)
                args = self.files(directory)
                for operation in (tool.export_evidence, tool.prepare_release):
                    with self.assertRaises(tool.EvidenceError):
                        operation(args)
                self.assertFalse(Path(args.output_dir).exists())
                self.assertFalse(Path(args.output).exists())

    def test_pending_export_is_evidence_not_capture_or_release(self):
        self.wf['status']['nodes']['example']['taskResultSynced'] = False
        with tempfile.TemporaryDirectory() as directory:
            args = self.files(directory)
            tool.export_evidence(args)
            exported = Path(args.output_dir)
            self.assertEqual(json.loads((exported / 'workflow.json').read_text()), self.wf)
            manifest = json.loads((exported / 'manifest.json').read_text())
            self.assertEqual(manifest['observation']['reason'], 'result-or-task-pending')
            self.assertFalse(manifest['automaticCaptureProved'])
            self.assertNotIn('capturedPodUID', json.loads((exported / 'nodes.json').read_text())['example'])
            with self.assertRaises(tool.EvidenceError):
                tool.prepare_release(args)
            self.assertFalse(Path(args.output).exists())

    def test_release_refuses_existing_file_or_symlink(self):
        for symlink in (False, True):
            with self.subTest(symlink=symlink), tempfile.TemporaryDirectory() as directory:
                args = self.files(directory)
                existing = Path(directory) / 'existing.json' if symlink else Path(args.output)
                existing.write_text('preserve existing evidence')
                if symlink:
                    Path(args.output).symlink_to(existing)
                with self.assertRaises(FileExistsError):
                    tool.prepare_release(args)
                self.assertEqual(existing.read_text(), 'preserve existing evidence')
                if symlink:
                    self.assertTrue(Path(args.output).is_symlink())

    def test_cli_inventory_and_failure_exit(self):
        with tempfile.TemporaryDirectory() as directory:
            args = self.files(directory)
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                code = tool.main(['inventory', '--pods', args.pod, '--workflows', args.workflow])
            self.assertEqual(code, 0)
            self.assertEqual(len(json.loads(output.getvalue())['pods']), 1)
            with contextlib.redirect_stderr(io.StringIO()):
                code = tool.main(['prepare-release', '--pod', args.pod, '--workflow', args.workflow, '--output', args.output])
            self.assertEqual(code, 2)
            self.assertFalse(Path(args.output).exists())

    def test_cli_accepts_null_stored_spec_without_claiming_capture(self):
        self.wf['status'].update(storedWorkflowSpec=None, storedTemplates=None)
        with tempfile.TemporaryDirectory() as directory:
            args = self.files(directory)
            commands = (['inventory', '--pods', args.pod, '--workflows', args.workflow],
                        ['export', '--pod', args.pod, '--workflow', args.workflow,
                         '--output-dir', args.output_dir],
                        ['prepare-release', '--pod', args.pod, '--workflow', args.workflow,
                         '--output', args.output, '--acknowledge-unproven-capture'])
            for command in commands:
                with self.subTest(command=command[0]):
                    output = io.StringIO()
                    with contextlib.redirect_stdout(output):
                        code = tool.main(command)
                    self.assertEqual(code, 0)
                    result = json.loads(output.getvalue())
                    row = result['pods'][0] if command[0] == 'inventory' else result['evidence']
                    self.assertEqual(row['reason'], 'legacy-no-receipt')
                    self.assertEqual(row['legacyClass'], 'memoization-transformation')
                    self.assertFalse(row['automaticCaptureProved'])
            self.assertFalse(result['applied'])
            self.assertEqual(json.loads((Path(args.output_dir) / 'workflow.json').read_text()), self.wf)
            self.assertEqual(json.loads(Path(args.output).read_text())[0],
                             {'op': 'test', 'path': '/metadata/uid', 'value': self.pod['metadata']['uid']})


if __name__ == '__main__':
    unittest.main(verbosity=2)

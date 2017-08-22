import logging
import sys
from ax.devops.utility.utilities import get_epoch_time_in_ms

logger = logging.getLogger(__name__)
logging.basicConfig(format="%(asctime)s.%(msecs)03d %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S",
                    level=logging.INFO,
                    stream=sys.stdout)


def get_retentions():
    return {
        'internal': 10000,
        'user-log': 20000,
        'ax-log': 30000,
        'exported': 40000,
    }


def get_fake_time(number=0):
    return get_epoch_time_in_ms() - number


def test_check_retention_1(artifactmanager, monkeypatch):
    logger.info('\nTest retention 1')
    artifact = {
      "archive_mode": "tar",
      "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
      "artifact_type": "internal",
      "ax_time": 1487643387690506,
      "ax_uuid": "c0432c6d-f7db-11e6-97d6-0a58c0a8840e",
      "ax_week": 2459,
      "checksum": "x7nolyJiEPxatr6T2LSYZQ==",
      "compression_mode": "gz",
      "deleted": 1,
      "deleted_by": "",
      "deleted_date": 0,
      "description": "ar description",
      "excludes": "null",
      "full_path": "cross-reference-step",
      "inline_storage": "",
      "is_alias": 1,
      "meta": {
        "ax_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "ax_timestamp": "1487643342732"
      },
      "name": "ar",
      "num_byte": 259076374,
      "num_dir": 442,
      "num_file": 2444,
      "num_other": 0,
      "num_skip": 0,
      "num_skip_byte": 0,
      "num_symlink": 15,
      "relative_path": "",
      "retention_tags": "exported",
      "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
      "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
      "src_name": "src",
      "src_path": "/",
      "storage_method": "s3",
      "storage_path": {
        "bucket": "test-cluster",
        "key": "test.code"
      },
      "stored_byte": 144682419,
      "structure_path": {
        "bucket": "test-cluster",
        "key": "test.structure"
      },
      "symlink_mode": "",
      "tags": '["test", "test2"]',
      "third_party": "",
      "timestamp": 1487643342732,
      "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set(),
                                                                    retention_policies=dict(),
                                                                    counter=0,
                                                                    dry_run=False)
    assert a is False
    assert b is 0
    assert c is 0
    assert d is 0
    assert e is None


def test_check_retention_2(artifactmanager, monkeypatch):
    logger.info('\nTest retention 2')
    monkeypatch.setattr(artifactmanager, '_delete_file_from_s3', lambda *args, **kwargs: True)
    artifact = {
        "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
        "artifact_type": "internal",
        "ax_time": 1487643387690506,
        "ax_uuid": "c0432c6d-f7db-11e6-97d6-0a58c0a8840e",
        "ax_week": 2459,
        "deleted": 0,
        "deleted_by": "",
        "deleted_date": 0,
        "description": "ar description",
        "full_path": "cross-reference-step",
        "inline_storage": "",
        "is_alias": 1,
        "name": "ar",
        "num_byte": 259076374,
        "relative_path": "",
        "retention_tags": "unknown",
        "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
        "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "src_name": "src",
        "src_path": "/",
        "storage_method": "s3",
        "stored_byte": 144682419,
        "third_party": "",
        "timestamp": 1487643342732,
        "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set('84b48169-f7db-11e6-8d5e-0a58c0a88407'),
                                                                    retention_policies=dict(),
                                                                    counter=0,
                                                                    dry_run=False)
    assert a is False
    assert b is 1
    assert c is 144682419
    assert d is 259076374
    assert e is None


def test_check_retention_3(artifactmanager, monkeypatch):
    logger.info('\nTest retention 3')
    monkeypatch.setattr(artifactmanager, '_delete_file_from_s3', lambda *args, **kwargs: True)
    artifact = {
        "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
        "artifact_type": "internal",
        "ax_time": 1487643387690506,
        "ax_uuid": "c0432c6d-f7db-11e6-97d6-0a58c0a8840e",
        "ax_week": 2459,
        "deleted": 0,
        "deleted_by": "",
        "deleted_date": 0,
        "description": "ar description",
        "full_path": "cross-reference-step",
        "inline_storage": "",
        "is_alias": 1,
        "name": "ar",
        "num_byte": 259076374,
        "relative_path": "",
        "retention_tags": "unknown",
        "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
        "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "src_name": "src",
        "src_path": "/",
        "stored_byte": 144682419,
        "storage_method": "s3",
        "third_party": "",
        "timestamp": 1487643342732,
        "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set(),
                                                                    retention_policies=dict(),
                                                                    counter=0,
                                                                    dry_run=False)
    assert a is False
    assert b is 1
    assert c is 144682419
    assert d is 259076374
    assert e is None


def test_check_retention_4(artifactmanager, monkeypatch):
    logger.info('\nTest retention 4')
    monkeypatch.setattr(artifactmanager, '_delete_file_from_s3', lambda *args, **kwargs: True)
    artifact = {
        "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
        "artifact_type": "internal",
        "deleted": 0,
        "full_path": "cross-reference-step",
        "is_alias": 1,
        "name": "ar",
        "num_byte": 259076374,
        "retention_tags": "exported",
        "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
        "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "src_name": "src",
        "src_path": "/",
        "stored_byte": 144682419,
        "storage_method": "s3",
        "tags": '["test", "test2"]',
        "timestamp": 1487643342732,
        "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set(),
                                                                    retention_policies=get_retentions(),
                                                                    counter=0,
                                                                    dry_run=False)
    assert a is False
    assert b is 1
    assert c is 144682419
    assert d is 259076374
    assert e is 'exported'


def test_check_retention_5(artifactmanager, monkeypatch):
    logger.info('\nTest retention 5')
    monkeypatch.setattr(artifactmanager, '_delete_file_from_s3', lambda *args, **kwargs: True)
    artifact = {
        "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
        "artifact_type": "internal",
        "deleted": 0,
        "full_path": "cross-reference-step",
        "is_alias": 1,
        "name": "ar",
        "num_byte": 259076374,
        "retention_tags": "exported",
        "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
        "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "src_name": "src",
        "src_path": "/",
        "stored_byte": 144682419,
        "storage_method": "s3",
        "tags": "",
        "timestamp": get_fake_time(100000),
        "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set(),
                                                                    retention_policies=get_retentions(),
                                                                    counter=0,
                                                                    dry_run=True)
    assert a is True
    assert b is 0
    assert c is 0
    assert d is 0
    assert e is 'exported'


def test_check_retention_6(artifactmanager, monkeypatch):
    logger.info('\nTest retention 6')
    monkeypatch.setattr(artifactmanager, '_delete_file_from_s3', lambda *args, **kwargs: True)
    artifact = {
        "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
        "artifact_type": "internal",
        "deleted": 2,
        "full_path": "cross-reference-step",
        "is_alias": 1,
        "name": "ar",
        "num_byte": 259076374,
        "retention_tags": "internal",
        "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
        "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "src_name": "src",
        "src_path": "/",
        "stored_byte": 144682419,
        "storage_method": "s3",
        "tags": "",
        "timestamp": get_fake_time(86400000*2),
        "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set(),
                                                                    retention_policies=get_retentions(),
                                                                    counter=0,
                                                                    dry_run=True)
    assert a is True
    assert b is 0
    assert c is 0
    assert d is 0
    assert e is 'internal'


def test_check_retention_7(artifactmanager, monkeypatch):
    logger.info('\nTest retention 7')
    monkeypatch.setattr(artifactmanager, '_delete_file_from_s3', lambda *args, **kwargs: True)
    artifact = {
        "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
        "artifact_type": "internal",
        "deleted": 0,
        "full_path": "cross-reference-step",
        "is_alias": 1,
        "name": "ar",
        "num_byte": 259076374,
        "retention_tags": "exported",
        "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
        "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "src_name": "src",
        "src_path": "/",
        "stored_byte": 144682419,
        "storage_method": "s3",
        "tags": "",
        "timestamp": get_fake_time(),
        "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set(),
                                                                    retention_policies=get_retentions(),
                                                                    counter=0,
                                                                    dry_run=True)
    assert a is False
    assert b is 1
    assert c is 144682419
    assert d is 259076374
    assert e is 'exported'


class AxdbFake(object):
    def update_artifact(self, payload):
        return True


def test_check_retention_8(artifactmanager, monkeypatch):
    logger.info('\nTest retention 8')
    monkeypatch.setattr(artifactmanager, '_delete_file_from_s3', lambda *args, **kwargs: True)

    artifact = {
        "artifact_id": "63308285-edb6-4d77-9061-60c90b6d6f07",
        "artifact_type": "internal",
        "ax_uuid": "c0432c6d-f7db-11e6-97d6-0a58c0a8840e",
        "deleted": 0,
        "full_path": "cross-reference-step",
        "is_alias": 1,
        "name": "ar",
        "num_byte": 259076374,
        "retention_tags": "internal",
        "service_instance_id": "84b488ae-f7db-11e6-8d5f-0a58c0a88407",
        "source_artifact_id": "0cbb6000-596d-4e69-80c4-fa09c981a5ef",
        "src_name": "src",
        "src_path": "/",
        "stored_byte": 144682419,
        "storage_method": "s3",
        "storage_path": '{"key": "test.key", "bucket": "test-cluster"}',
        "tags": "",
        "timestamp": get_fake_time(100000),
        "workflow_id": "84b48169-f7db-11e6-8d5e-0a58c0a88407"
    }
    a, b, c, d, e = artifactmanager.check_artifact_retention_policy(artifact=artifact,
                                                                    live_workflows=set(),
                                                                    retention_policies=get_retentions(),
                                                                    counter=0,
                                                                    dry_run=False)
    assert a is True
    assert b is 0
    assert c is 0
    assert d is 0
    assert e is 'internal'

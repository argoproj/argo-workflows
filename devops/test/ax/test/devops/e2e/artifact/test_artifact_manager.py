# Copyright 2015-2017 Applatix, Inc.  All rights reserved.

import logging
import random
import time
import uuid
from concurrent.futures import ThreadPoolExecutor, as_completed

from ax.devops.artifact.constants import RETENTION_TAGS
from ax.devops.utility.utilities import random_text, retry_on_errors

logger = logging.getLogger(__name__)


def test_artifact_creation(artifact_manager, concurrency, max_request):
    """Stress test artifact creation API.

    :param artifact_manager:
    :param concurrency:
    :param max_request:
    :returns:
    """

    def create_artifact(sn):
        """Create an artifact

        :param sn:
        :returns:
        """
        logger.info('Creating artifact (sn: %s) ...', sn)
        try:
            num_byte = random.randint(0, 1024 ** 3)
            stored_byte = random.randint(0, num_byte)
            artifact = {
                "artifact_id": str(uuid.uuid1()),
                "service_instance_id": str(uuid.uuid1()),
                "full_path": None,
                "name": random_text(8),
                "description": random_text(16),
                "storage_method": "s3",
                "storage_path": None,
                "num_byte": num_byte,
                "num_dir": 0,
                "num_file": 1,
                "num_other": 0,
                "num_skip_byte": 0,
                "num_skip": 0,
                "compression_mode": "",
                "archive_mode": "",
                "stored_byte": stored_byte,
                "meta": None,
                "timestamp": int(time.time()),
                "workflow_id": str(uuid.uuid1()),
                "checksum": random_text(32, lower_or_upper=True),
                "tags": '[]',
                "retention_tags": random.choice(list(RETENTION_TAGS)),
                "deleted": 0,
            }
            response = artifact_manager.create_artifact(artifact, max_retry=20, value_only=True,
                                                        retry_on_exception=retry_on_errors(errors=['ERR_API_INVALID_PARAM'], retry=False))
        except Exception as e:
            logger.error('Failed to create artifact (sn: %s): %s', sn, str(e))
        else:
            logger.info('Successfully created artifact (sn: %s)', sn)
            return response

    start_time = time.time()

    count = 0
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = []
        for i in range(max_request):
            futures.append(executor.submit(create_artifact, i))
        for future in as_completed(futures):
            if future.result():
                count += 1

    end_time = time.time()
    logger.info('Totally created %s artifacts in %s seconds', count, end_time - start_time)

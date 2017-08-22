# Copyright 2015-2017 Applatix, Inc.  All rights reserved.

import logging
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

logger = logging.getLogger(__name__)
logging.getLogger('requests.packages.urllib3.connectionpool').setLevel(logging.ERROR)


def test_get_artifacts(axdb, concurrency, max_request, artifact_id):
    """Stress test against artifact search API

    :param axdb:
    :param concurrency:
    :param max_request:
    :param artifact_id:
    :returns:
    """

    def get_artifacts(sn):
        """Get artifacts

        :param sn:
        :returns:
        """
        logger.info('Retrieving artifacts (sn: %s) ...', sn)
        start_time = time.time()
        try:
            axdb.get_artifacts({'artifact_id': artifact_id})
        except Exception as e:
            logger.error('Failed to retrieve artifacts (sn: %s): %s', sn, str(e))
            return False
        else:
            end_time = time.time()
            logger.info('Successfully retrieved artifacts (sn: %s) in %s seconds', sn, end_time - start_time)
            return True

    start_time = time.time()

    count = 0
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = []
        for i in range(max_request):
            futures.append(executor.submit(get_artifacts, i))
        for future in as_completed(futures):
            if future.result():
                count += 1

    end_time = time.time()
    logger.info('Totally spent %s seconds to process %s requests', end_time - start_time, count)

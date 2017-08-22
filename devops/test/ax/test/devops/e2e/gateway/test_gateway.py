# Copyright 2015-2017 Applatix, Inc.  All rights reserved.

import logging
import requests
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

logger = logging.getLogger(__name__)
logging.getLogger('requests.packages.urllib3.connectionpool').setLevel(logging.ERROR)


def test_get_branches(gateway, concurrency, max_request):
    """Stress test against branch API on gateway

    :param gateway:
    :param concurrency:
    :param max_request:
    :returns:
    """

    def get_branches(sn):
        """Get branches

        :param sn:
        :returns:
        """
        logger.info('Retrieving branches (sn: %s) ...', sn)
        start_time = time.time()
        try:
            resp = requests.get('{}/v1/scm/branches'.format(gateway))
        except Exception as e:
            logger.error('Failed to retrieve branches (sn: %s): %s', sn, str(e))
        else:
            branches = resp.json()['data']
            end_time = time.time()
            logger.info('Successfully retrieved %s branches (sn: %s) in %s seconds', len(branches), sn, end_time - start_time)
            return branches

    start_time = time.time()

    count = 0
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = []
        for i in range(max_request):
            futures.append(executor.submit(get_branches, i))
        for future in as_completed(futures):
            if future.result():
                count += 1

    end_time = time.time()
    logger.info('Totally spent %s seconds to process %s requests', end_time - start_time, count)

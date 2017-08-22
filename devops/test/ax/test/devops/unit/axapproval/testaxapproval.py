from ax.util.az_patch import az_patch
az_patch()

import unittest
import os
from ax.devops.axapproval.axapproval import AXApproval, AXApprovalException
from unittest.mock import patch


class TestAXApproval(unittest.TestCase):
    def test_task_id_not_set(self):
        if 'AX_CONTAINER_NAME' in os.environ:
            del os.environ['AX_CONTAINER_NAME']
        with self.assertRaises(AXApprovalException) as cm:
            AXApproval(None, None, None, None)
        self.assertEqual(
            'AX_CONTAINER_NAME cannot be found in the container ENV.',
            str(cm.exception)
        )

    def test_none_list(self):
        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'
        with self.assertRaises(AXApprovalException) as cm:
            AXApproval(None, None, None, None)
        self.assertEqual(
            'required_list and optional_list cannot both be None.',
            str(cm.exception)
        )
        del os.environ['AX_CONTAINER_NAME']

    def test_negative_number(self):
        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'
        with self.assertRaises(AXApprovalException) as cm:
            AXApproval("user1", "", -1, -1)
        self.assertEqual(
            'number_optional or timeout cannot be negative.',
            str(cm.exception)
        )
        del os.environ['AX_CONTAINER_NAME']

    def test_incorrect_optional(self):
        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'
        with self.assertRaises(AXApprovalException) as cm:
            AXApproval("user1", "", 1, 1)
        self.assertEqual(
            'number_optional cannot be greater than optional_list.',
            str(cm.exception)
        )
        with self.assertRaises(AXApprovalException) as cm:
            AXApproval("", "user1,user2", 3, 1)
        self.assertEqual(
            'number_optional cannot be greater than optional_list.',
            str(cm.exception)
        )
        del os.environ['AX_CONTAINER_NAME']

    def test_duplicate_list(self):
        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'
        with self.assertRaises(AXApprovalException) as cm:
            AXApproval("user1", "user1", 1, 1)
        self.assertEqual(
            "{'user1'} cannot be in both required_list and optional_list.",
            str(cm.exception)
        )
        with self.assertRaises(AXApprovalException) as cm:
            AXApproval("user2, user3", "user1,user2", 2, 1)
        self.assertEqual(
            "{'user2'} cannot be in both required_list and optional_list.",
            str(cm.exception)
        )
        del os.environ['AX_CONTAINER_NAME']

    # Mock and functionalities using external API
    @patch('ax.devops.axapproval.axapproval.AXApproval.notification')
    @patch('ax.devops.axapproval.axapproval.AXApproval.retrieve_redis')
    def test_success(self, mock_retrieve_redis, mock_notification):
        mock_notification.return_value = True
        mock_retrieve_redis.return_value = [{'user': 'user1', 'result': True, 'timestamp': '201667123'}]

        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'

        axapproval = AXApproval("user1", "", 0, 1)
        with self.assertRaises(SystemExit) as cm:
            axapproval.run()

        self.assertEqual(cm.exception.code, 0)
        del os.environ['AX_CONTAINER_NAME']

    @patch('ax.devops.axapproval.axapproval.AXApproval.notification')
    @patch('ax.devops.axapproval.axapproval.AXApproval.retrieve_redis')
    def test_success2(self, mock_retrieve_redis, mock_notification):
        mock_notification.return_value = True
        mock_retrieve_redis.return_value = [{'user': 'user1', 'result': True, 'timestamp': '201667123'},
                                            {'user': 'user2', 'result': True, 'timestamp': '201667123'}]

        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'

        axapproval = AXApproval("user1", "user2,user3", 1, 1)
        with self.assertRaises(SystemExit) as cm:
            axapproval.run()

        self.assertEqual(cm.exception.code, 0)
        del os.environ['AX_CONTAINER_NAME']

    @patch('ax.devops.axapproval.axapproval.AXApproval.notification')
    @patch('ax.devops.axapproval.axapproval.AXApproval.retrieve_redis')
    def test_success3(self, mock_retrieve_redis, mock_notification):
        mock_notification.return_value = True
        mock_retrieve_redis.return_value = [{'user': 'user1', 'result': True, 'timestamp': '201667123'},
                                            {'user': 'user2', 'result': True, 'timestamp': '201667123'},
                                            {'user': 'user3', 'result': True, 'timestamp': '201667123'},
                                            {'user': 'user4', 'result': True, 'timestamp': '201667123'},
                                            {'user': 'user5', 'result': True, 'timestamp': '201667123'},
                                            {'user': 'user6', 'result': True, 'timestamp': '201667123'}]

        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'

        axapproval = AXApproval("user1,user2,user3", "user4,user5", 2, 1)
        with self.assertRaises(SystemExit) as cm:
            axapproval.run()

        self.assertEqual(cm.exception.code, 0)
        del os.environ['AX_CONTAINER_NAME']

    @patch('ax.devops.axapproval.axapproval.AXApproval.notification')
    @patch('ax.devops.axapproval.axapproval.AXApproval.retrieve_redis')
    def test_fail1(self, mock_retrieve_redis, mock_notification):
        mock_notification.return_value = True
        mock_retrieve_redis.return_value = None

        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'

        axapproval = AXApproval("user1", "user2,user3", 1, 1)
        with self.assertRaises(AXApprovalException) as cm:
            axapproval.run()

        self.assertEqual(
            "Timeout for getting approvals. Exit.",
            str(cm.exception)
        )
        del os.environ['AX_CONTAINER_NAME']

    @patch('ax.devops.axapproval.axapproval.AXApproval.notification')
    @patch('ax.devops.axapproval.axapproval.AXApproval.retrieve_redis')
    def test_fail2(self, mock_retrieve_redis, mock_notification):
        mock_notification.return_value = True
        mock_retrieve_redis.return_value = [{'user': 'user1', 'result': True, 'timestamp': '201667123'}]

        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'

        axapproval = AXApproval("user1", "user2,user3", 1, 1)
        with self.assertRaises(AXApprovalException) as cm:
            axapproval.run()

        self.assertEqual(
            "Timeout for getting approvals. Exit.",
            str(cm.exception)
        )
        del os.environ['AX_CONTAINER_NAME']

    @patch('ax.devops.axapproval.axapproval.AXApproval.notification')
    @patch('ax.devops.axapproval.axapproval.AXApproval.retrieve_redis')
    def test_fail3(self, mock_retrieve_redis, mock_notification):
        mock_notification.return_value = True
        mock_retrieve_redis.return_value = [{'user': 'user1', 'result': True, 'timestamp': '201667123'},
                                            {'user': 'user2', 'result': True, 'timestamp': '201667123'}]

        if 'AX_CONTAINER_NAME' not in os.environ:
            os.environ['AX_CONTAINER_NAME'] = 'axapproval-1234-1234-1234-1234'

        axapproval = AXApproval("user1", "user2,user3", 2, 1)
        with self.assertRaises(AXApprovalException) as cm:
            axapproval.run()

        self.assertEqual(
            "Timeout for getting approvals. Exit.",
            str(cm.exception)
        )
        del os.environ['AX_CONTAINER_NAME']

if __name__ == '__main__':
    unittest.main()

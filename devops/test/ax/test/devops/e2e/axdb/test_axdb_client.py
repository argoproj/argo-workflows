
from ax.util.az_patch import az_patch
az_patch()

import logging
import unittest
import os
from ax.devops.axdb.axdb_client import AxdbClient

# This is for testing the axdb wrapper. Currently it is more like a integration test which requires
# a testing axdb server to run the test. One can run the test by specify the following env
# parameters in their docker options
#
# axdb_host_for_test
# axdb_port_for_test


class TestAXDBClient(unittest.TestCase):
    axdb_client = None

    @classmethod
    def setUpClass(cls):
        # Setting up AXDB server for testing
        host = os.getenv('axdb_host_for_test', 'axdb.axsys')
        port = os.getenv('axdb_port_for_test', 8083)
        cls.axdb_client = AxdbClient(host=host, port=port, version='v1')
        cls.axdb_client.create_table(cls.axdb_client.tables['workflow']['schema'])
        # cls.axdb_client.delete_table(cls.axdb_client.tables['workflow_leaf_service']['name'])
        cls.axdb_client.create_table(cls.axdb_client.tables['workflow_leaf_service']['schema'])
        cls.axdb_client.delete_workflow_status('test123')
        cls.axdb_client.delete_leaf_service_result('testleaf123')
        cls.axdb_client.delete_leaf_service_result('testleaf234')
        cls.axdb_client.delete_leaf_service_result('testleaf345')

    @classmethod
    def tearDownClass(cls):
        cls.axdb_client.delete_workflow_status('test123')
        cls.axdb_client.delete_leaf_service_result('testleaf123')
        cls.axdb_client.delete_leaf_service_result('testleaf234')
        cls.axdb_client.delete_leaf_service_result('testleaf345')

    def testGetNonExistedWorkflow(self):
        self.assertEqual(self.axdb_client.get_workflow_status('test123'), None)

    def testGetExistedWorkflow(self):
        self.axdb_client.create_workflow_status('test123', 'DONE', None, 9876)
        self.assertEqual(self.axdb_client.get_workflow_status('test123'), {'id': 'test123', 'status': 'DONE', 'service_template': '', 'timestamp': 9876})
        self.axdb_client.delete_workflow_status('test123')

    def testCreateWorkflow(self):
        self.axdb_client.delete_workflow_status('test123')
        result = self.axdb_client.create_workflow_status('test123', 'DONE', 'A template', 9876)
        self.assertTrue(result)
        self.assertEqual(self.axdb_client.get_workflow_status('test123'), {'id': 'test123', 'status': 'DONE', 'service_template': 'A template', 'timestamp': 9876})
        self.axdb_client.delete_workflow_status('test123')

    def testCreateDupWorkflow(self):
        self.axdb_client.delete_workflow_status('test123')
        result = self.axdb_client.create_workflow_status('test123', 'DONE', 'A template', 9876)
        self.assertTrue(result)
        result = self.axdb_client.create_workflow_status('test123', 'FAIL', 'A template', 9877)
        self.assertEqual(result, False)
        self.assertEqual(self.axdb_client.get_workflow_status('test123'), {'id': 'test123', 'status': 'DONE', 'service_template': 'A template', 'timestamp': 9876})
        self.axdb_client.delete_workflow_status('test123')

    def testDeleteWorkflow(self):
        self.axdb_client.create_workflow_status('test123', 'DONE', 'Another template', 9876)
        result = self.axdb_client.delete_workflow_status('test123')
        self.assertTrue(result)
        self.assertEqual(self.axdb_client.get_workflow_status('test123'), None)

    def testUpdateWorkflow(self):
        self.axdb_client.create_workflow_status('test123', 'DONE', 'A template', 9876)
        result = self.axdb_client.update_workflow_status('test123', {'status': 'FAIL', 'timestamp': 9877})
        self.assertTrue(result)
        self.assertEqual(self.axdb_client.get_workflow_status('test123'), {'id': 'test123', 'status': 'FAIL', 'service_template': 'A template', 'timestamp': 9877})
        self.axdb_client.delete_workflow_status('test123')

    def testUpdateConditionalWorkflowStatus(self):
        self.axdb_client.create_workflow_status('test123', 'DONE', 'A template', 9876)
        result = self.axdb_client.update_conditional_workflow_status('test123', 9877, 'FAIL', 'DONE')
        self.assertTrue(result)
        self.assertEqual(self.axdb_client.get_workflow_status('test123'), {'id': 'test123', 'status': 'FAIL', 'service_template': 'A template', 'timestamp': 9877})
        result = self.axdb_client.update_conditional_workflow_status('test123', 'FAIL', 'DONE', 9878)
        self.assertFalse(result)
        self.axdb_client.delete_workflow_status('test123')

    def testDeleteNonExistWorkflow(self):
        result = self.axdb_client.delete_workflow_status('test1234')
        self.assertTrue(result)

    def testCreateDuplicateWorkflow(self):
        self.axdb_client.create_workflow_status('test123', 'DONE', 'A template', 9876)
        self.assertFalse(self.axdb_client.create_workflow_status('test123', 'DONE', 'A template', 9875))
        self.axdb_client.delete_workflow_status('test123')

    def testGetCertainColumnWorkflow(self):
        self.axdb_client.create_workflow_status('test123', 'DONE', 'A template', 9876)
        result = self.axdb_client.get_workflow_certain_columns('test123', ['status', 'timestamp'])
        self.assertEqual(result, {'status': 'DONE', 'timestamp': 9876})
        result = self.axdb_client.get_workflow_certain_columns('test123', ['status', 'service_template'])
        self.assertEqual(result, {'status': 'DONE', 'service_template': 'A template'})
        self.axdb_client.delete_workflow_status('test123')

    def testGetLeafServiceResult(self):
        self.axdb_client.create_leaf_service_result('testleaf123', 'testC', 0, 'FAIL', 9876)
        self.assertEqual(self.axdb_client.get_leaf_service_result_by_leaf_id('testleaf123'), [{'leaf_id': 'testleaf123', 'root_id': 'testC', 'sn': 0, 'result': 'FAIL', 'detail': '', 'timestamp': 9876}])
        self.axdb_client.delete_leaf_service_result('testleaf123')

    def testGetLeafServiceResults(self):
        self.axdb_client.create_leaf_service_result('testleaf123', 'testA', 0, 'FAIL', 9876)
        self.axdb_client.create_leaf_service_result('testleaf123', 'testA', 1, 'FAIL', 9877)
        self.axdb_client.create_leaf_service_result('testleaf123', 'testA', 2, 'FAIL', 9878)
        self.axdb_client.create_leaf_service_result('testleaf124', 'testA', 3, 'FAIL', 9878)
        self.assertEqual(len(self.axdb_client.get_leaf_service_results('testA')), 4)
        self.axdb_client.delete_leaf_service_result('testleaf123')
        self.axdb_client.delete_leaf_service_result('testleaf124')
        self.assertEqual(len(self.axdb_client.get_leaf_service_results('testA')), 0)

    def testGetLeafServiceResults2(self):
        self.axdb_client.create_leaf_service_result('testleaf123B', 'testB', 0, 'FAIL', 9876)
        self.axdb_client.create_leaf_service_result('testleaf124B', 'testB', 0, 'FAIL', 9877)
        self.axdb_client.create_leaf_service_result('testleaf125B', 'testB', 0, 'FAIL', 9878)
        self.assertEqual(len(self.axdb_client.get_leaf_service_results('testB')), 1)
        self.axdb_client.delete_leaf_service_result('testleaf123B')
        self.axdb_client.delete_leaf_service_result('testleaf124B')
        self.axdb_client.delete_leaf_service_result('testleaf125B')
        self.assertEqual(len(self.axdb_client.get_leaf_service_results('testB')), 0)

# Basic logging.
logging.basicConfig(format="%(asctime)s %(levelname)s %(name)s %(lineno)d %(threadName)s: %(message)s")
logging.getLogger("ax").setLevel(logging.DEBUG)

if __name__ == '__main__':
    unittest.main()

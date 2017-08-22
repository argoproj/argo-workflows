import logging
import time
import threading
from ax.devops.artifact.artifact_cache import ArtifactCache, ArtifactFileObject

logger = logging.getLogger(__name__)


def fake_read_1_sec(self, file_name):
    logger.info("start fake_read_1_sec for file: %s", file_name)
    time.sleep(1)
    logger.info("finish fake_read_1_sec for file: %s", file_name)


def fake_read_2_sec(self, file_name):
    logger.info("start fake_read_2_sec for file: %s", file_name)
    time.sleep(2)
    logger.info("finish fake_read_2_sec for file: %s", file_name)


def fake_read_4_sec(self, file_name):
    logger.info("start fake_read_4_sec for file: %s", file_name)
    time.sleep(4)
    logger.info("finish fake_read_4_sec for file: %s", file_name)


def fake_read_8_sec(self, file_name):
    logger.info("start fake_read_8_sec for file: %s", file_name)
    time.sleep(8)
    logger.info("finish fake_read_8_sec for file: %s", file_name)


def fake_read_16_sec(self, file_name):
    logger.info("start fake_read_16_sec for file: %s", file_name)
    time.sleep(16)
    logger.info("finish fake_read_16_sec for file: %s", file_name)


def fake_write_1_sec(self):
    logger.info("start fake_write_1_sec for file")
    time.sleep(1)
    logger.info("finish fake_write_1_sec for file")


def fake_write_2_sec(self):
    logger.info("start fake_write_2_sec for file")
    time.sleep(2)
    logger.info("finish fake_write_2_sec for file")


def fake_write_4_sec(self):
    logger.info("start fake_write_4_sec for file")
    time.sleep(4)
    logger.info("finish fake_write_4_sec for file")


def fake_write_8_sec(self):
    logger.info("start fake_write_8_sec for file")
    time.sleep(8)
    logger.info("finish fake_write_8_sec for file")


def fake_write_16_sec(self):
    logger.info("start fake_write_16_sec for file")
    time.sleep(16)
    logger.info("finish fake_write_16_sec for file")


def fake_delete_1_sec(self):
    logger.info("start fake_delete_1_sec for file")
    time.sleep(1)
    logger.info("finish fake_delete_1_sec for file")


def fake_delete_2_sec(self):
    logger.info("start fake_delete_2_sec for file")
    time.sleep(2)
    logger.info("finish fake_delete_2_sec for file")


def fake_delete_4_sec(self):
    logger.info("start fake_delete_4_sec for file")
    time.sleep(4)
    logger.info("finish fake_delete_4_sec for file")


def fake_delete_8_sec(self):
    logger.info("start fake_delete_8_sec for file")
    time.sleep(8)
    logger.info("finish fake_delete_8_sec for file")


def fake_delete_16_sec(self):
    logger.info("start fake_delete_16_sec for file")
    time.sleep(16)
    logger.info("finish fake_delete_16_sec for file")


def test_logic_sequential_1(monkeypatch):
    monkeypatch.setattr(ArtifactFileObject, 'read_file', fake_read_1_sec)
    monkeypatch.setattr(ArtifactFileObject, 'write_file', fake_write_1_sec)
    monkeypatch.setattr(ArtifactFileObject, 'remove_file', fake_delete_1_sec)

    my_cache = ArtifactCache(10)
    my_cache.get_file('k1', 4, '/f1', '/s3/f1', 't1')
    my_cache.get_file('k2', 6, '/f2', '/s3/f2', 't2')

    assert len(my_cache.key_dict) == len(my_cache.key_queue) == 2
    assert my_cache.key_queue[0].key == 'k1'
    assert my_cache.key_queue[1].key == 'k2'


# def test_logic_sequential_2(monkeypatch):
#     monkeypatch.setattr(ArtifactFileObject, 'read_file', fake_read_1_sec)
#     monkeypatch.setattr(ArtifactFileObject, 'write_file', fake_write_1_sec)
#     monkeypatch.setattr(ArtifactFileObject, 'remove_file', fake_delete_1_sec)
#
#     my_cache = ArtifactCache(10)
#     my_cache.get_file('k1', 4, '/f1', '/s3/f1', 't1')
#     my_cache.get_file('k2', 6, '/f2', '/s3/f2', 't2')
#     my_cache.get_file('k3', 5, '/f3', '/s3/f3', 't3')
#
#     assert len(my_cache.key_dict) == len(my_cache.key_queue) == 1
#     assert my_cache.key_queue[0].key == 'k3'
#
#
# def test_logic_sequential_3(monkeypatch):
#     monkeypatch.setattr(ArtifactFileObject, 'read_file', fake_read_1_sec)
#     monkeypatch.setattr(ArtifactFileObject, 'write_file', fake_write_1_sec)
#     monkeypatch.setattr(ArtifactFileObject, 'remove_file', fake_delete_1_sec)
#
#     my_cache = ArtifactCache(10)
#     my_cache.get_file('k1', 4, '/f1', '/s3/f1', 't1')
#     my_cache.get_file('k2', 6, '/f2', '/s3/f2', 't2')
#     my_cache.get_file('k3', 3, '/f3', '/s3/f3', 't3')
#     my_cache.get_file('k4', 2, '/f4', '/s3/f4', 't4')
#
#     assert len(my_cache.key_dict) == len(my_cache.key_queue) == 2
#     assert my_cache.key_queue[0].key == 'k3'
#     assert my_cache.key_queue[1].key == 'k4'


def test_logic_parallel_read_different_files(monkeypatch):
    monkeypatch.setattr(ArtifactFileObject, 'read_file', fake_read_4_sec)
    monkeypatch.setattr(ArtifactFileObject, 'write_file', fake_write_1_sec)
    monkeypatch.setattr(ArtifactFileObject, 'remove_file', fake_delete_1_sec)

    my_cache = ArtifactCache(10)
    my_cache.get_file('k1', 4, '/f1', '/s3/f1', 't1')
    my_cache.get_file('k2', 6, '/f2', '/s3/f2', 't2')

    start_time = time.time()
    t1 = threading.Thread(name="Thread-1", target=my_cache.get_file, args=('k1', 4, '/f1', '/s3/f1', 't1'))
    t1.daemon = True

    t2 = threading.Thread(name="Thread-2", target=my_cache.get_file, args=('k2', 6, '/f2', '/s3/f2', 't2'))
    t2.daemon = True

    t1.start()
    t2.start()
    t1.join()
    t2.join()
    end_time = time.time()

    assert end_time - start_time < 4500  # guaranteed parallel


def test_logic_parallel_write_different_files(monkeypatch):
    monkeypatch.setattr(ArtifactFileObject, 'read_file', fake_read_1_sec)
    monkeypatch.setattr(ArtifactFileObject, 'write_file', fake_write_4_sec)
    monkeypatch.setattr(ArtifactFileObject, 'remove_file', fake_delete_1_sec)

    my_cache = ArtifactCache(10)
    my_cache.get_file('k1', 4, '/f1', '/s3/f1', 't1')
    my_cache.get_file('k2', 6, '/f2', '/s3/f2', 't2')

    start_time = time.time()
    t1 = threading.Thread(name="Thread-1", target=my_cache.get_file, args=('k3', 3, '/f3', '/s3/f3', 't3'))
    t1.daemon = True

    t2 = threading.Thread(name="Thread-2", target=my_cache.get_file, args=('k4', 7, '/f4', '/s3/f4', 't4'))
    t2.daemon = True

    t1.start()
    t2.start()
    t1.join()
    t2.join()
    end_time = time.time()

    assert end_time - start_time < 5500  # guaranteed parallel
    assert len(my_cache.key_dict) == len(my_cache.key_queue) == 2
    assert my_cache.key_queue[0].key == 'k3'
    assert my_cache.key_queue[1].key == 'k4'



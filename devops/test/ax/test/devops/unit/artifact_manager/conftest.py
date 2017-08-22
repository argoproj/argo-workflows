import logging
import pytest
import sys


logging.basicConfig(format="%(asctime)s.%(msecs)03d %(levelname)s %(name)s %(threadName)s: %(message)s",
                    datefmt="%Y-%m-%dT%H:%M:%S",
                    level=logging.INFO,
                    stream=sys.stdout)
logger = logging.getLogger(__name__)

class FakeAxdbClient(object):
    def update_artifact(self, payload):
        return True

@pytest.fixture
def artifactmanager():
    from ax.devops.artifact.artifactmanager import ArtifactManager
    artifact_manager = ArtifactManager()
    artifact_manager.axdb_client = FakeAxdbClient()
    return artifact_manager
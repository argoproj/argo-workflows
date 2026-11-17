"""Exercise release metadata using real Git repositories and Go binaries.

Run: python3 -m unittest discover -s hack -p 'build_metadata_test.py' -v
"""
import os
import re
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parent


class BuildMetadataTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.source = self.root / "repo"
        self.source.mkdir()
        self.env = dict(os.environ, GOWORK="off", GOPROXY="off", GOSUMDB="off",
                        GOTOOLCHAIN="local", GOFLAGS="", GIT_CONFIG_NOSYSTEM="1",
                        GIT_CONFIG_GLOBAL=os.devnull)
        self.run_cmd("git", "init", "--template=", str(self.source))
        self.run_cmd("git", "config", "user.name", "Metadata Test")
        self.run_cmd("git", "config", "user.email", "test@example.invalid")
        (self.source / "go.mod").write_text(
            "module github.com/argoproj/argo-workflows/v4\n\ngo 1.24.0\n")
        (self.source / "main.go").write_text("package main\nfunc main() {}\n")
        (self.source / "Dockerfile").write_text("FROM scratch\n")
        (self.source / ".dockerignore").write_text(".git\n*.md\n")
        (self.source / "README.md").write_text("tracked source\n")
        self.run_cmd("git", "add", ".")
        self.run_cmd("git", "commit", "-m", "fixture")

    def run_cmd(self, *args, cwd=None, ok=True):
        result = subprocess.run(args, cwd=cwd or self.source, env=self.env,
                                text=True, capture_output=True, timeout=300)
        if ok:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout

    def prepare(self):
        target = self.root / "context"
        self.run_cmd("bash", str(SCRIPTS / "prepare-release-context.sh"), str(target))
        return target / "source"

    def build(self, source, vcs="true"):
        binary = self.root / "binary"
        self.run_cmd("go", "build", "-buildvcs=" + vcs, "-o", str(binary), ".", cwd=source)
        return binary

    def verify(self, source, binary, ok=True):
        self.run_cmd("bash", str(SCRIPTS / "verify-build-metadata.sh"),
                     str(binary), cwd=source, ok=ok)

    def test_without_git_reproduces_devel(self):
        exported = self.root / "exported"
        shutil.copytree(self.source, exported, ignore=shutil.ignore_patterns(".git"))
        binary = self.build(exported, "auto")
        info = self.run_cmd("go", "version", "-m", str(binary))
        self.assertIn("(devel)", info)
        self.verify(self.source, binary, ok=False)

    def test_release_tag_and_sanitized_context(self):
        self.run_cmd("git", "tag", "-a", "v4.1.6", "-m", "release")
        self.run_cmd("git", "config", "http.extraheader", "test-only-secret")
        (self.source / ".git/hooks").mkdir()
        (self.source / ".git/hooks/post-checkout").write_text("untrusted hook")
        (self.source / "untracked-secret").write_text("must not be exported")
        source = self.prepare()
        self.assertFalse((source / "untracked-secret").exists())
        self.assertFalse((source / ".git/hooks/post-checkout").exists())
        self.assertNotIn("test-only-secret", (source / ".git/config").read_text())
        self.assertTrue((source / "README.md").exists())
        self.assertEqual((source.parent / "Dockerfile.dockerignore").read_text(), "!**\n")
        binary = self.build(source)
        self.verify(source, binary)
        self.assertIn("\tv4.1.6", self.run_cmd("go", "version", "-m", str(binary)))

    def test_untagged_commit_uses_pseudo_version(self):
        source = self.prepare()
        self.verify(source, self.build(source))

    def test_lightweight_tag_and_linux_binary(self):
        self.run_cmd("git", "tag", "v4.1.6")
        self.env.update(GOOS="linux", GOARCH="amd64", CGO_ENABLED="0")
        source = self.prepare()
        self.verify(source, self.build(source))

    def test_non_utc_commit_time(self):
        self.env.update(GIT_COMMITTER_DATE="2026-11-17T12:00:00-05:00")
        self.run_cmd("git", "commit", "--allow-empty", "-m", "timezone")
        source = self.prepare()
        self.verify(source, self.build(source))

    def test_untagged_descendant_retains_release_base(self):
        self.run_cmd("git", "tag", "v4.1.6")
        self.run_cmd("git", "commit", "--allow-empty", "-m", "after release")
        source = self.prepare()
        binary = self.build(source)
        self.verify(source, binary)
        self.assertIn("v4.1.7-0.", self.run_cmd("go", "version", "-m", str(binary)))

    def test_large_inventory_with_submodule_is_rejected(self):
        revision = self.run_cmd("git", "rev-parse", "HEAD").strip()
        self.run_cmd("git", "update-index", "--add", "--cacheinfo",
                     "160000", revision, "000-submodule")
        for i in range(1000):
            (self.source / f"file-{i:04d}").write_text("fixture")
        self.run_cmd("git", "add", "file-*")
        self.run_cmd("git", "commit", "-m", "submodule fixture")
        self.run_cmd("bash", str(SCRIPTS / "prepare-release-context.sh"),
                             str(self.root / "context"), ok=False)

    def test_highest_semver_tag_wins(self):
        self.run_cmd("git", "tag", "-a", "v4.1.6", "-m", "old tag")
        self.run_cmd("git", "tag", "v4.1.7")
        source = self.prepare()
        binary = self.build(source)
        self.verify(source, binary)
        self.assertIn("\tv4.1.7", self.run_cmd("go", "version", "-m", str(binary)))

    @unittest.skipUnless(os.environ.get("ARGO_METADATA_DOCKER_TEST") == "1",
                         "set ARGO_METADATA_DOCKER_TEST=1 to exercise Docker contexts")
    def test_docker_context_preserves_git_metadata(self):
        builder = re.search(r"^FROM (golang:\S+) AS builder$",
                            (SCRIPTS.parent / "Dockerfile").read_text(), re.M).group(1)
        (self.source / "Dockerfile").write_text(
            f"FROM {builder}\nRUN apk add --no-cache git bash\n"
            "WORKDIR /src\nCOPY . .\n"
            "RUN go build -buildvcs=true -o /binary .\n")
        self.run_cmd("git", "add", "Dockerfile")
        self.run_cmd("git", "commit", "-m", "Docker fixture")
        self.run_cmd("git", "tag", "v4.1.6")
        source = self.prepare()
        tag = "argo-metadata-test:" + self.root.name.lower()
        self.addCleanup(subprocess.run, ["docker", "image", "rm", "-f", tag],
                        capture_output=True)
        self.run_cmd("docker", "build", "-t", tag, str(self.source))
        before = self.run_cmd("docker", "run", "--rm", tag,
                              "go", "version", "-m", "/binary")
        self.assertIn("(devel)", before)
        self.run_cmd("docker", "build", "-t", tag, "-f",
                     str(source.parent / "Dockerfile"), str(source))
        after = self.run_cmd("docker", "run", "--rm", tag,
                             "go", "version", "-m", "/binary")
        self.assertIn("\tv4.1.6", after)
        self.assertIn("vcs.modified=false", after)

    def test_dirty_source_is_rejected(self):
        (self.source / "main.go").write_text("package main\nfunc main() { println(1) }\n")
        self.run_cmd("bash", str(SCRIPTS / "prepare-release-context.sh"),
                     str(self.root / "context"), ok=False)
        self.verify(self.source, self.build(self.source), ok=False)

    def test_wrong_revision_is_rejected(self):
        source = self.prepare()
        binary = self.build(source)
        (self.source / "README.md").write_text("next revision\n")
        self.run_cmd("git", "add", ".")
        self.run_cmd("git", "commit", "-m", "next")
        self.verify(self.source, binary, ok=False)


if __name__ == "__main__":
    unittest.main()

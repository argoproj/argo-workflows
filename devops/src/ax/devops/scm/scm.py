import abc
import logging
import os
import re
import shlex
import subprocess
import tempfile
import threading
import time
from urllib.parse import quote, urlparse

from ax.devops.exceptions import AXScmException
from ax.devops.utility.utilities import utc
from ax.exceptions import AXTimeoutException

logger = logging.getLogger('ax.devops.scm')

__all__ = [
    'ScmClient',
    'GitClient',
    'CodeCommitClient',
    'HgClient',
]

DEFAULT_CMD_TIMEOUT = 30 * 60
MAGIC_HASH = '4b825dc642cb6eb9a060e54bf8d69288fbee4904'  # This is a special hash indicating empty commit


class ScmClient(object):
    def __init__(self, path, repo=None, username=None, password=None, cmd_timeout=None, enable_cache=False):
        """Client to remote SCM repository

        :param repo: remote repository
        :param path: workspace
        :param username: username to repository
        :param password: password to repository
        :param enable_cache:
        """
        self.path = path
        # Default to https if repo does not specify protocol scheme
        if repo:
            parsed_url = urlparse(repo, scheme='https')
            self.repo_url = parsed_url.geturl()
        else:
            self.repo_url = None
        self.username = username
        self.password = password
        self.env = os.environ.copy()
        self.cmd_timeout = DEFAULT_CMD_TIMEOUT if cmd_timeout is None else cmd_timeout
        self.enable_cache = enable_cache

    @abc.abstractmethod
    def clone(self, commit=None, branch=None):
        """Clone the repository into a local path and optionally checkout a commit or branch

        :param commit: commit to clone
        :param branch: branch to clone
        """

    @abc.abstractmethod
    def merge(self, branch, message=None, author=None):
        """Merge a branch into local workspace

        :param branch: branch to merge in
        :param message: message to use as the merge commit message
        :param author: author used in commit
        """

    @abc.abstractmethod
    def push(self):
        """Push local commits to remote"""

    def run_cmd(self, cmd, shell=False, retry=None, retry_interval=None, timeout=None, binary_mode=False, **kwargs):
        """Wrapper around subprocess.Popen to capture/print output run the command with cwd to local repo

        :param cmd:
        :param shell: execute the command in a shell
        :param retry: number of retries to perform if command fails with non-zero return code
        :param retry_interval: interval between retries
        :param timeout: timeout in seconds before failing
        :param binary_mode: return result in binary mode
        """
        orig_cmd = cmd
        output = ""
        if not shell:
            cmd = shlex.split(cmd)
        attempts = 1 if not retry else 1 + retry
        retry_interval = retry_interval or 10
        cwd = kwargs.pop('cwd', self.path)
        env = kwargs.pop('env', self.env)
        if timeout is None:
            timeout = self.cmd_timeout

        def timeout_proc(tproc, timeout, result):
            """Helper to kill a process after some time"""
            try:
                tproc.wait(timeout)
                timed_out = False
            except subprocess.TimeoutExpired:
                tproc.terminate()
                timed_out = True
            result['timed_out'] = timed_out
            return timed_out

        timed_out_res = {}
        err_reason = None
        for attempt in range(attempts):
            logger.info('$ %s', orig_cmd)
            proc = subprocess.Popen(cmd, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, shell=shell, **kwargs)
            if timeout:
                timed_out_res = {}
                t = threading.Thread(target=timeout_proc, args=(proc, timeout, timed_out_res))
                t.start()
            if binary_mode:
                output = proc.stdout.read()
            else:
                lines = []
                for line in iter(proc.stdout.readline, b''):
                    line = line.decode(errors='replace')
                    if line[-1] == '\n':
                        line = line[0:-1]  # AA-2093: Should only strip off last character if it is a newline
                    logger.debug(line)
                    lines.append(line)
                output = '\n'.join(lines)
            proc.stdout.close()
            proc.wait()
            if proc.returncode == 0:
                break
            else:
                if timed_out_res.get('timed_out'):
                    err_reason = "command '{}' timed out ({}s)".format(orig_cmd, timeout)
                else:
                    err_reason = "command '{}' failed with return code {}".format(orig_cmd, proc.returncode)

                if attempt + 1 < attempts:
                    err_reason = "Attempt %s/%s of %s. Retrying in %ss" % (attempt + 1, attempts, err_reason, retry_interval)
                    logger.warning(err_reason)
                    time.sleep(retry_interval)
                else:
                    logger.warning(err_reason)
        else:
            if output:
                logger.error(output)
            if timed_out_res.get('timed_out'):
                raise AXTimeoutException("{} timed out ({}s)".format(orig_cmd, timeout))
            else:
                raise AXScmException(err_reason)
        return output

    @staticmethod
    def _merge_message(branch):
        return "Merge branch '{}'".format(branch)


class GitClient(ScmClient):
    def __init__(self, *args, **kwargs):
        self.use_permanent_credentials = kwargs.pop('use_permanent_credentials', False)
        self.read_only = kwargs.pop('read_only', False)
        super(GitClient, self).__init__(*args, **kwargs)
        if self.use_permanent_credentials:
            self.credentials_file = '{}/.git/credential'.format(self.path)
        else:
            self.credentials_file = None
        if not self.read_only:
            self.init()
        else:
            self.repo_url = self.get_remote()

    def set_credentials(self):
        # Public repository
        if not self.username or not self.password:
            return
        protocol, split_url = re.split('://', self.repo_url)
        insecure_repo_url = '{}://{}:{}@{}'.format(protocol, quote(self.username), quote(self.password), split_url)
        # Use permanent credential file
        if self.use_permanent_credentials:
            self._set_permanent_credentials(insecure_repo_url)
        # Use permanent credential file
        else:
            self._set_temporary_credentials(insecure_repo_url)

    def _set_permanent_credentials(self, url):

        def get_credentials():
            if os.path.isfile(self.credentials_file):
                with open(self.credentials_file, 'r') as f:
                    url = f.readline().strip()
                    m = re.match(r'(.*)://(.*):(.*)@(.*)/(.*)/(.*)', url)
                    if m:
                        return m.groups()[1], m.groups()[2]

        # If credential is out-dated, remove it
        credential = get_credentials()
        if credential and credential != (self.username, self.password):
            self._remove_permanent_credentials()

        # Create permanent credential
        if not credential or credential != (self.username, self.password):
            with open(self.credentials_file, 'w+') as f:
                f.write(url + '\n')
            self.run_cmd("git config --local credential.helper 'store --file={}'".format(self.credentials_file))

    def _set_temporary_credentials(self, url):
        if not self.credentials_file:
            self.credentials_file = tempfile.NamedTemporaryFile(mode='w')
            self.credentials_file.write(url + '\n')
            self.credentials_file.flush()
            self.run_cmd("git config --local credential.helper 'store --file={}'".format(self.credentials_file.name))

    def remove_credentials(self):
        # Public repository
        if not self.username or not self.password:
            return
        # Use permanent credential file
        if self.use_permanent_credentials:
            self._remove_permanent_credentials()
        # Use permanent credential file
        else:
            self._remove_temporary_credentials()

    def _remove_permanent_credentials(self):
        self.run_cmd("git config --local --remove-section credential")
        if os.path.isfile(self.credentials_file):
            os.remove(self.credentials_file)

    def _remove_temporary_credentials(self):
        if not self.credentials_file:
            return
        self.run_cmd("git config --local --remove-section credential")
        self.credentials_file.close()
        self.credentials_file = None

    def set_remote(self):
        if not self.get_remote():
            self.run_cmd('git remote add origin {}'.format(self.repo_url))

    def get_remote(self):
        try:
            output = self.run_cmd('git remote get-url origin').strip()
        except AXScmException:
            return
        else:
            return output

    def init(self):
        """Initialize local repository, set credentials, and add remote"""
        if not os.path.isdir(self.path):
            os.makedirs(self.path)
        if not os.path.isdir(os.path.join(self.path, '.git')):
            self.run_cmd("git init --template /dev/null {}".format(self.path))

        # If user did not supply repository and we cannot infer repository
        # from workspace, throw exception. If we can infer repository from
        # workspace, use the referred repository.
        old_repo_url = self.get_remote()
        if not self.repo_url:
            if not old_repo_url:
                msg = 'Repo URL is required, as it cannot be inferred from supplied workspace ({})'.format(self.path)
                logger.error(msg)
                raise AXScmException('Missing repo URL', detail=msg)
            else:
                self.repo_url = old_repo_url

        # If user supplies repository and the repository can actually
        # be inferred from workspace, we need to compare these two values.
        # If the input repository is different from the inferred repository,
        # we consider it as an illegal situation and will raise exception.
        if old_repo_url and old_repo_url != self.repo_url:
            msg = 'Supplied repo URL ({}) is different from repo URL inferred from workspace ({})'.format(self.repo_url, old_repo_url)
            logger.error(msg)
            raise AXScmException('Inconsistent repo URL', detail=msg)

        self.set_credentials()
        self.set_remote()

    def fetch(self):
        self.run_cmd('git fetch -p')

    def list_remote(self, heads=False, tags=False, repo=None, pattern=None):
        """List references in a remote repository

        :param heads: limit to heads
        :param tags: limit to tags
        :param repo:
        :param pattern:
        """
        cmd_args = ['git', 'ls-remote']
        if heads:
            cmd_args.append('--heads')
        if tags:
            cmd_args.append('--tags')
        if repo:
            cmd_args.append(repo)
        if pattern:
            cmd_args.append(pattern)
        cmd = ' '.join(cmd_args)
        out = self.run_cmd(cmd)
        refs = []
        for line in out.splitlines():
            if line.startswith('From'):
                continue
            try:
                commit, reference = line.split()
            except ValueError:
                continue
            else:
                refs.append({'commit': commit, 'reference': reference})
        return refs

    def list_tree(self, branch=None, commit=None, subdir=''):
        """Run `git ls-tree` command.

        :param branch:
        :param commit:
        :param subdir:
        :return:
        """
        if branch:
            ref = 'refs/remotes/origin/{}'.format(branch)
        elif commit:
            ref = commit
        else:
            ref = 'refs/remotes/origin/master'
        cmd = 'git ls-tree -r -l {} {}'.format(ref, subdir)
        out = self.run_cmd(cmd, retry=5, retry_interval=5)
        tree = []
        for line in out.splitlines():
            mode, git_type, sha, size, path = line.split()
            tree.append({
                'mode': mode,
                'type': git_type,
                'sha': sha,
                'size': size,
                'path': path
            })
        return tree

    def get_files(self, branch=None, commit=None, subdir='', binary_mode=False, filter_yaml=False):
        """Retrieve files under a subdir.

        :param branch:
        :param commit:
        :param subdir:
        :param binary_mode:
        :return:
        """
        tree = self.list_tree(branch, commit, subdir)
        files = []
        for item in tree:
            if filter_yaml:
                # Make sure the files are actual yaml files.
                if not item['path'].endswith('yml') and not item['path'].endswith('yaml'):
                    continue
            content = self.run_cmd('git show {}'.format(item['sha']), binary_mode=binary_mode)
            files.append({
                'path': item['path'],
                'content': content
            })
        return files

    def is_binary_file(self, path, branch=None, commit=None):
        """Determine if a file is a binary file

        :param path:
        :param branch:
        :param commit:
        :returns:
        """
        if branch:
            ref = 'refs/remotes/origin/{}'.format(branch)
        elif commit:
            ref = commit
        else:
            ref = 'refs/remotes/origin/master'
        output = self.run_cmd('git diff {} --numstat {} -- {} | cut -f1'.format(MAGIC_HASH, ref, path))
        return bool(output == '-')

    def get_remote_heads(self):
        """Get all remote heads

        :return:
        """
        cmd = 'git for-each-ref refs/remotes/origin --format="%(refname:strip=3) %(objectname) %(committerdate:raw)"'
        out = self.run_cmd(cmd)
        refs = []
        for line in out.splitlines():
            reference, commit, commit_date, _ = line.split()
            refs.append({'commit': commit, 'reference': reference, 'commit_date': int(commit_date)})
        return refs

    def get_remote_head(self, branch):
        """Get the commit for the remote head of a branch

        :param branch:
        :return:
        """
        cmd = 'git for-each-ref refs/remotes/origin/{} --format="%(objectname)"'.format(branch)
        out = self.run_cmd(cmd)
        return out if out else None

    def get_commit_branches(self, commit):
        """Get branches containing the commit.

        :param commit:
        :return:
        """
        cmd = 'git branch --all --contains {}'.format(commit)
        out = self.run_cmd(cmd)
        branches = sorted([line.split('/', maxsplit=2)[-1] for line in out.splitlines()])
        return branches

    def _construct_commit_object(self, lines_of_a_commit, having_line_break=True):
        """Construct a commit object.

        :param lines_of_a_commit:
        :param having_line_break: Only the last entry won't have a line break.
        :return:
        """
        commit = {
            'repo': self.repo_url
        }
        if len(lines_of_a_commit) < 6:
            logger.warning('Illegal texts forming the contents of a commit, skip processing')
            for i in range(len(lines_of_a_commit)):
                print(lines_of_a_commit[i])
            return
        for i in range(len(lines_of_a_commit)):
            if lines_of_a_commit[i].startswith('commit'):
                if '(' in lines_of_a_commit[i]:
                    m = re.match('commit (.*) \((.*)\)', lines_of_a_commit[i])
                    if m:
                        revisions, refs = m.groups()
                        revisions = revisions.split()
                        refs = [v.split('/', maxsplit=1)[-1] for v in refs.split(', ')]
                        commit['revision'] = revisions[0]
                        commit['branches'] = refs
                        commit['parents'] = revisions[1:]
                    else:
                        logger.warning('Unable to match commit with pattern')
                else:
                    m = re.match('commit (.*)', lines_of_a_commit[i])
                    if m:
                        revisions = m.groups()[0]
                        revisions = revisions.split()
                        commit['revision'] = revisions[0]
                        commit['branches'] = []
                        commit['parents'] = revisions[1:]
                    else:
                        logger.warning('Unable to match commit with pattern')
            elif lines_of_a_commit[i].startswith('Author:'):
                commit['author'] = lines_of_a_commit[i].split(':')[-1].strip()
            elif lines_of_a_commit[i].startswith('AuthorDate:'):
                author_date = lines_of_a_commit[i].split(':', maxsplit=1)[-1].strip()
                author_date = utc(author_date, type=int)
                commit['author_date'] = author_date
            elif lines_of_a_commit[i].startswith('Commit:'):
                commit['committer'] = lines_of_a_commit[i].split(':')[-1].strip()
            elif lines_of_a_commit[i].startswith('CommitDate:'):
                commit_date = lines_of_a_commit[i].split(':', maxsplit=1)[-1].strip()
                commit_date = utc(commit_date, type=int)
                commit['commit_date'] = commit_date
                commit['date'] = commit_date
            elif lines_of_a_commit[i].startswith('Merge:'):
                continue
            else:
                if having_line_break:
                    description = '\n'.join([v.lstrip() for v in lines_of_a_commit[i + 1:-1]])
                else:
                    description = '\n'.join([v.lstrip() for v in lines_of_a_commit[i + 1:]])
                commit['description'] = description
                break
        return commit

    def get_commits(self, branch=None, commit=None, since=None, until=None, author=None, committer=None, description=None, limit=None):
        """Get commits on a branch after sha.

        :param branch:
        :param commit:
        :param since:
        :param until:
        :param author:
        :param committer:
        :param description:
        :param limit: if supplied, retrieve up to the number of commits
        :return:
        """
        if commit:
            cmd = ('git log {commit} {since} {until} {limit} --date=iso-strict '
                   '--pretty=fuller --decorate --parents --date-order').format(
                commit=commit,
                since='--since={}'.format(since) if since else '',
                until='--until={}'.format(until) if until else '',
                limit='--max-count=1'
            )
        else:
            if not branch:
                branch_condition = '--remotes'
            elif type(branch) in {list, tuple}:
                branch_conditions = []
                for i in range(len(branch)):
                    branch_conditions.append('remotes/origin/{}'.format(branch[i]))
                branch_condition = ' '.join(branch_conditions)
            else:
                branch_condition = 'remotes/origin/{}'.format(branch)

            cmd = ('git log {branch} {since} {until} {author} {committer} {description} {limit} '
                   '--date=iso-strict --pretty=fuller --decorate --parents --date-order').format(
                branch=branch_condition,
                since='--since={}'.format(since) if since else '',
                until='--until={}'.format(until) if until else '',
                author='--author="{}"'.format(author) if author else '',
                committer='--committer="{}"'.format(committer) if committer else '',
                description='--grep="{}"'.format(description) if description else '',
                limit='--max-count={}'.format(limit) if limit else ''
            )

        try:
            proc = subprocess.Popen(shlex.split(cmd), cwd=self.path, env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            lines_of_a_commit = []
            for line in iter(proc.stdout.readline, b''):
                line = line.decode(errors='replace')
                # If line starts with 'commit', it is time to construct a commit object
                if line.startswith('commit') and len(lines_of_a_commit) > 0:
                    commit = self._construct_commit_object(lines_of_a_commit)
                    lines_of_a_commit = []
                    if commit:
                        yield commit
                lines_of_a_commit.append(line)
            if len(lines_of_a_commit) > 0:
                commit = self._construct_commit_object(lines_of_a_commit, having_line_break=False)
                if commit:
                    yield commit
            proc.stdout.close()
            proc.wait()
        except subprocess.CalledProcessError:
            raise AXScmException('Command failed', detail='Command ({}) failed'.format(cmd))

    def get_commit(self, sha):
        """Get a single commit.

        :param sha:
        :return:
        """
        # Get all information except branch
        cmd = 'git show {} --quiet --date=iso-strict --pretty=fuller --parents --quiet --decorate'.format(sha)
        output = self.run_cmd(cmd).split('\n')
        commit = self._construct_commit_object(output, having_line_break=False)

        # Get branch information
        branches = self.get_commit_branches(sha)
        commit['branches'] = branches
        return commit

    @property
    def author(self):
        if not hasattr(self, '_author'):
            author = self.username or 'argouser'
            author_email = '{}@argoproj'.format(author)
            self._author = "{} <{}>".format(author, author_email)
        return self._author

    def clone(self, commit=None, branch=None):
        # NOTE: replaced `git clone` with `git init`, `git remote`, `git fetch` in order to store credentials locally
        # self.run_cmd("git clone --template /dev/null {} {}".format(self.repo_url, path))
        self.init()
        self.run_cmd('git fetch --tags --progress')
        out = self.run_cmd("git remote set-head origin -a")
        origin_head = re.search(r"^origin/HEAD set to (.*)", out).group(1)
        self.run_cmd("git checkout {}".format(commit or branch or origin_head))
        if not self.is_clean():
            raise AXScmException('Workspace unclean')

    def get_current_branch(self):
        return self.run_cmd("git rev-parse --abbrev-ref HEAD").strip()

    def get_current_commit(self):
        return self.run_cmd("git rev-parse HEAD").strip()

    def merge(self, branch, message=None, author=None):
        author = author or self.author
        if message is None:
            message = self._merge_message(branch)
        if self.is_branch(branch):
            branch = 'origin/{}'.format(branch)
        output = self.run_cmd('git merge --no-commit --no-ff {}'.format(branch))
        if re.match(r"^Already up-to-date.", output):
            return
        self.run_cmd('git commit -m "{}" --author "{}"'.format(message, author))

    def push(self):
        self.run_cmd("git push")

    def is_branch(self, string):
        try:
            self.run_cmd('git show-ref --verify refs/remotes/origin/{}'.format(string))
        except AXScmException:
            return False
        else:
            return True

    def is_commit(self, string):
        try:
            self.run_cmd('git rev-parse --verify "{}^{{commit}}"'.format(string))
        except AXScmException:
            return False
        else:
            return True

    def is_commit_in_branch(self, commit, branch):
        common_ancestor = self.run_cmd('git merge-base "{}" "{}"'.format(commit, branch)).strip()
        return common_ancestor == commit

    def is_clean(self):
        """Test if current workspace is clean or not."""
        output = self.run_cmd('git diff --shortstat').strip()
        return not output


class CodeCommitClient(GitClient):
    def __init__(self, *args, **kwargs):
        GitClient.__init__(self, *args, **kwargs)
        if self.username and self.password:
            # AWS CLI will look to environment variables first (before looking at ~/.aws/credentials)
            # This avoids us having to modify configuration files
            self.env['AWS_ACCESS_KEY_ID'] = self.username
            self.env['AWS_SECRET_ACCESS_KEY'] = self.password
        self.init()

    def set_credentials(self):
        """Set git's credential helper for the local repo to use aws codecommit credential-helper CLI"""
        # See:
        #   http://docs.aws.amazon.com/codecommit/latest/userguide/setting-up-https-unixes.html
        #   http://docs.aws.amazon.com/codecommit/latest/userguide/how-to-connect.html#how-to-connect-prerequisites
        # Use --local instead of --global to avoid modifying global environment.
        if not self.username or not self.password:
            return
        self.run_cmd("git config --local credential.helper '!aws codecommit credential-helper $@'")
        self.run_cmd("git config --local credential.UseHttpPath true")


class HgClient(ScmClient):
    def clone(self, commit=None, branch=None):
        if self.username and self.password:
            protocol, split_url = re.split('://', self.repo_url)
            repo_url = '{}://{}:{}@{}'.format(protocol, quote(self.username), quote(self.password), split_url)
        else:
            repo_url = self.repo_url
        cmd = "hg clone {} {}".format(repo_url, self.path)
        if commit or branch:
            cmd += ' --rev {}'.format(commit or branch)
        logger.info(cmd.replace(self.password, '******') if self.password else cmd)
        self.run_cmd(cmd)

    def merge(self, *args, **kwargs):
        raise NotImplementedError("not implemented")

    def push(self, *args, **kwargs):
        raise NotImplementedError("not implemented")

import copy
import datetime
import jira
import json
import jwt
import logging
import os
import requests
import time

from concurrent.futures import ThreadPoolExecutor, as_completed
from flask import Flask, request, jsonify, make_response, Response, redirect
from werkzeug.exceptions import BadRequest
from urllib.parse import unquote, urlparse

from .gateway import Gateway
from .constants import AxEventTypes, ScmVendors, SUPPORTED_TYPES
from .event_translators import EventTranslator
from ax.devops.jira.jira_utils import translate_jira_issue_event
from ax.devops.kafka.kafka_client import ProducerClient
from ax.devops.redis.redis_client import RedisClient, DB_RESULT
from ax.devops.settings import AxSettings
from ax.devops.utility.utilities import AxPrettyPrinter, top_k, sort_str_dictionaries
from ax.exceptions import AXException, AXIllegalArgumentException, AXIllegalOperationException, \
    AXApiResourceNotFound, AXApiInvalidParam, AXApiInternalError
from ax.notification_center import CODE_JOB_CI_STATUS_REPORTING_FAILURE, CODE_JOB_CI_EVENT_CREATION_FAILURE, \
    CODE_JOB_CI_YAML_UPDATE_FAILURE, CODE_PLATFORM_ERROR


logger = logging.getLogger(__name__)

app = Flask(__name__)
gateway = None


def get_json():
    """Helper to retrieve json from the request body, or raise AXApiInvalidParam if invalid"""
    try:
        return request.get_json(force=True)
    except Exception:
        raise AXApiInvalidParam("Invalid json supplied")


# For some reason, BadRequest is not handled by the generic Exception error handler, so both decorators are needed
@app.errorhandler(400)
@app.errorhandler(Exception)
def error_handler(error):
    # Our exceptions and error handling is a complete mess. Need to clean this up and standardize across teams
    if isinstance(error, AXException):
        data = error.json()
        if isinstance(error, (AXIllegalArgumentException, AXApiInvalidParam, AXIllegalOperationException)):
            status_code = 400
        elif isinstance(error, AXApiResourceNotFound):
            status_code = 404
        else:
            logger.exception("Internal error")
            status_code = 500
    else:
        if isinstance(error, BadRequest):
            logger.exception("Bad request")
            code = AXApiInvalidParam.code
            status_code = error.code
        else:
            logger.exception("Internal error")
            code = "ERR_AX_INTERNAL"
            status_code = 500
        data = {"code": code,
                "message": str(error),
                "detail": ""}
    logger.warning('%s (status_code: %s): %s', error, status_code, data)
    return make_response(jsonify(data), status_code)


@app.route('/v1/ping', methods=['GET'])
def ping():
    return Response('"pong"', mimetype='application/json')


@app.route('/v1/scm/test', methods=['POST'])
def test():
    """Test connection to SCM server."""
    payload = get_json()
    scm_type = payload.get('type', '').lower()
    url = payload.get('url', '').lower()
    username = payload.get('username', None)
    password = payload.get('password', None)
    logger.info('Received request (type: %s, url: %s, username: %s, password: ******)', scm_type, url, username)
    if not scm_type:
        raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (type)')
    if scm_type not in SUPPORTED_TYPES:
        raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(scm_type))
    if scm_type == ScmVendors.GIT:
        assert url, AXApiInvalidParam('Missing required parameters',
                                      detail='Require parameter (url) when type is {}'.format(ScmVendors.GIT))
    else:
        assert all([username, password]), AXApiInvalidParam('Missing required parameters',
                                                            detail='Required parameters (username, password, url)')
    try:
        repos = gateway.get_repos(scm_type=scm_type, url=url, username=username, password=password)
    except Exception as e:
        logger.warning('Failed to get repositories: %s', e)
        raise AXApiInternalError('Failed to get repositories', detail=str(e))
    else:
        return jsonify({'repos': repos})


@app.route('/v1/scm/events', methods=['POST'])
def events():
    """Create a DevOps event."""
    payload, headers = get_json(), request.META
    try:
        logger.info('Translating SCM event ...')
        event_list = EventTranslator.translate(payload, headers)
    except Exception as e:
        logger.error('Failed to translate event: %s', e)
        # Todo Tianhe Issue: #330 comment out for now because it is distracting
        # gateway.event_notification_client.send_message_to_notification_center(
        #     CODE_JOB_CI_EVENT_TRANSLATE_FAILURE,
        #     detail={'payload': payload, 'error': str(e)})
        raise AXApiInternalError('Failed to translate event', detail=str(e))
    else:
        logger.info('Successfully translated event')

    kafka_client = ProducerClient()
    successful_events = []
    for event in event_list:
        if event['type'] == AxEventTypes.PING:
            logger.info('Received a PING event, skipping service creation ...')
            continue
        else:
            try:
                logger.info('Creating AX event ...\n%s', AxPrettyPrinter().pformat(event))
                key = '{}_{}_{}'.format(event['repo'], event['branch'], event['commit'])
                kafka_client.send(AxSettings.TOPIC_DEVOPS_CI_EVENT, key=key, value=event, timeout=120)
            except Exception as e:
                gateway.event_notification_client.send_message_to_notification_center(
                    CODE_JOB_CI_EVENT_CREATION_FAILURE,
                    detail={'event_type': event.get('type', 'UNKNOWN'),
                            'error': str(e)})
                logger.warning('Failed to create AX event: %s', e)
            else:
                logger.info('Successfully created AX event')
                successful_events.append(event)
    kafka_client.close()
    return Response(successful_events)


@app.route('/v1/scm/reports', methods=['POST'])
def reports():
    """Report build/test status to source control tool."""
    payload = get_json()
    logger.info('Received reporting request (payload: %s)', payload)
    report_id = payload.get('id')
    repo = payload.get('repo')
    if not report_id:
        raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (id)')

    try:
        if not repo:
            cache = gateway.redis_client.get(report_id, decoder=json.loads)
            repo = cache['repo']
        vendor = gateway.axops_client.get_tool(repo)['type']
        if vendor not in gateway.scm_clients.keys():
            raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(vendor))
        result = gateway.scm_clients[vendor].upload_job_result(payload)
        if result == -1:
            logger.info('GitHub does not support status report for the non-sha commits. Skip.')
    except Exception as e:
        logger.error('Failed to report status: %s', e)
        gateway.event_notification_client.send_message_to_notification_center(
            CODE_JOB_CI_STATUS_REPORTING_FAILURE, detail=payload)
        raise AXApiInternalError('Failed to report status', detail=str(e))
    else:
        logger.info('Successfully reported status')
        return jsonify(result)


@app.route('/v1/scm/webhooks', methods=['GET', 'POST', 'DELETE'])
def get_create_delete_webhooks():
    """Create / delete a webhook."""
    payload = get_json()

    repo = payload.get('repo')
    vendor = payload.get('type')
    username = payload.get('username')
    password = payload.get('password')
    if not all([repo, vendor]):
        raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (repo, type)')
    if vendor not in gateway.scm_clients.keys():
        raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(vendor))

    if username and password:
        gateway.scm_clients[vendor].update_repo_info(repo, vendor, username, password)
    if request.method == 'GET':
        result = gateway.get_webhook(vendor, repo)
    elif request.method == 'POST':
        result = gateway.create_webhook(vendor, repo)
    else:
        result = gateway.delete_webhook(vendor, repo)
    return jsonify(result)


@app.route('/v1/scm/yamls', methods=['POST'])
def post_yamls():
    """Update YAML contents (i.e. policy, template)."""
    payload = get_json()

    vendor = payload.get('type')
    repo = payload.get('repo')
    branch = payload.get('branch')
    if not all([vendor, repo, branch]):
        raise AXApiInvalidParam('Missing required parameters', detail='Required parameters (type, repo, branch)')
    if vendor not in gateway.scm_clients.keys():
        raise AXApiInvalidParam('Invalid parameter values', detail='Unsupported type ({})'.format(vendor))

    try:
        # The arrival of events may not always be in the natural order of commits. For
        # example, the user may resent an old event from UI of source control tool. In
        # this case, we may update the YAML contents to an older version. To avoid this,
        # we guarantee that every YAML update will only update the content to the latest
        # version on a branch. More specifically, whenever we receive an event, we extract
        # the repo and branch information, and find the HEAD of the branch. Then, we use
        # the commit of HEAD to retrieve the YAML content, and update policies/templates
        # correspondingly.
        scm_client = gateway.scm_clients[vendor]
        commit = scm_client.get_branch_head(repo, branch)
        yaml_files = scm_client.get_yamls(repo, commit)
        logger.info('Updating YAML contents (policy/template) ...')
        gateway.axops_client.update_yamls(repo, branch, commit, yaml_files)
    except Exception as e:
        logger.error('Failed to update YAML contents: %s', e)
        gateway.event_notification_client.send_message_to_notification_center(
            CODE_JOB_CI_YAML_UPDATE_FAILURE, detail={'vendor': vendor,
                                                     'repo': repo,
                                                     'branch': branch,
                                                     'error': str(e)})
        raise AXApiInternalError('Failed to update YAML contents', str(e))
    else:
        logger.info('Successfully updated YAML contents')
        return jsonify({})


@app.route('/v1/scm/branches', methods=['GET', 'DELETE'])
def get_delete_branches():
    """Query branches."""
    repo = request.args.get('repo')
    branch = request.args.get('branch', "") or request.args.get('name', "")
    if request.method == 'DELETE':
        gateway.purge_branches(repo, branch)
        return jsonify({})
    else:
        if branch and branch.startswith('~'):
            branch = branch[1:]
        order_by = request.args.get('order_by', "")
        limit = request.args.get('limit', "")
        if limit:
            limit = int(limit)
        branches = gateway.get_branches(repo, branch, order_by, limit)
        return jsonify({'data': branches})


@app.route('/v1/scm/commits', methods=['GET', 'DELETE'])
def get_commits():
    """Query commits."""
    # Repo and branch are optional parameters that can always be used to reduce
    # search scope. Repo is used to construct the path to the workspace so that
    # the number of commands we issue can be significantly reduced. Branch can
    # be used in every command to filter commits by reference (branch).
    repo = request.args.get('repo')
    branch = request.args.get('branch')
    repo_branch = request.args.get('repo_branch')
    if repo_branch and (repo or branch):
        raise AXApiInvalidParam('Ambiguous query condition',
                                'It is ambiguous to us to supply both repo_branch and repo/branch')
    workspaces = gateway._parse_repo_branch(repo, branch, repo_branch)

    # If commit / revision is supplied, we will disrespect all other parameters.
    # Also, we no longer use `git log` to issue query but use `git show` to directly
    # show the commit information.
    commit = request.args.get('commit') or request.args.get('revision')

    # Full-text search can be performed against 3 fields: author, committer, and description.
    # To perform narrow search, specify `author=~<author>&committer=~<committer>&description=~<description>`.
    # To perform broad search, specify `search=~<search>`.
    # Note that, in git, all queries are full-text search already, so we will strip off `~`.
    search = request.args.get('search')
    author = request.args.get('author', None)
    committer = request.args.get('committer', None)
    description = request.args.get('description', None)
    if search:
        use_broad_search = True
    else:
        use_broad_search = False
    if author:
        author = author.split(',')
    else:
        author = [None]
    if committer:
        committer = committer.split(',')
    else:
        committer = [None]
    author_committer = []
    for i in range(len(author)):
        for j in range(len(committer)):
            author_committer.append([author[i], committer[j]])

    # We use time-based pagination. min_time is converted to since and max_time is
    # converted to until. Also, the time format seconds since epoch (UTC).
    since = request.args.get('min_time')
    until = request.args.get('max_time')
    if since:
        since = datetime.datetime.utcfromtimestamp(int(since)).strftime('%Y-%m-%dT%H:%M:%S')
    if until:
        until = datetime.datetime.utcfromtimestamp(int(until)).strftime('%Y-%m-%dT%H:%M:%S')

    # Limit specify the maximal records that we return. Fields specify the fields
    # that we return. Sort allows the sorting of final results.
    limit = request.args.get('limit')
    fields = request.args.get('fields')
    sorter = request.args.get('sort')
    if limit:
        limit = int(limit)
    if fields:
        fields = set(fields.split(','))
    if sorter:
        sorters = sorter.split(',')
        valid_keys = {'repo', 'revision', 'author', 'author_date', 'committer', 'commit_date', 'date', 'description'}
        valid_sorters = []
        for i in range(len(sorters)):
            key = sorters[i][1:] if sorters[i].startswith('-') else sorters[i]
            if key in valid_keys:
                valid_sorters.append(sorters[i])
        sorter = valid_sorters

    logger.info('Retrieving commits (repo: %s, branch: %s, commit: %s, limit: %s) ...', repo, branch, commit, limit)

    # Prepare arguments for workspace scanning
    search_conditions = []
    for key in workspaces.keys():
        if not os.path.isdir(key):  # If the workspace does not exist, we should skip scanning it
            continue
        elif commit:
            search_conditions.append({'workspace': key, 'commit': commit})
        elif use_broad_search:
            for j in range(len(author_committer)):
                _author, _committer = author_committer[j][0], author_committer[j][1]
                _search_dict = {'workspace':   key,
                                'branch':      list(workspaces[key]),
                                'since':       since,
                                'until':       until,
                                'limit':       limit,
                                'author':      _author,
                                'committer':   _committer,
                                'description': description,
                                }
                for field in {'author', 'committer', 'description'}:
                    new_dict = copy.deepcopy(_search_dict)
                    new_dict[field] = search
                    search_conditions.append(new_dict)
        else:
            for j in range(len(author_committer)):
                _author, _committer = author_committer[j][0], author_committer[j][1]
                search_conditions.append({'workspace': key, 'branch': list(workspaces[key]),
                                          'author': _author, 'committer': _committer, 'description': description,
                                          'since': since, 'until': until, 'limit': limit})

    # Scan workspaces
    commits_list = []
    with ThreadPoolExecutor(max_workers=20) as executor:
        futures = []
        for i in range(len(search_conditions)):
            if commit:
                futures.append(executor.submit(Gateway._get_commit, **search_conditions[i]))
            else:
                futures.append(executor.submit(Gateway._get_commits, **search_conditions[i]))
        for future in as_completed(futures):
            try:
                data = future.result()
                if data:
                    commits_list.append(data)
            except Exception as e:
                logger.warning('Unexpected exception occurred during processing: %s', e)

    if commit:
        # If commit is supplied in the query, the return list is a list of commits,
        # so we do not need to run top_k algorithm
        top_commits = sorted(commits_list, key=lambda v: -v['date'])
    else:
        # Retrieve top k commits
        top_commits = top_k(commits_list, limit, key=lambda v: -v['date'])

    # Sort commits
    if sorter:
        top_commits = sort_str_dictionaries(top_commits, sorter)
    else:
        top_commits = sorted(top_commits, key=lambda v: -v['date'])

    # Filter fields
    for i in range(len(top_commits)):
        for k in list(top_commits[i].keys()):
            if fields is not None and k not in fields:
                del top_commits[i][k]
    logger.info('Successfully retrieved commits')

    return jsonify({'data': top_commits})


@app.route('/v1/scm/commit/<pk>', methods=['GET'])
def get_commit(pk):
    """Get a single commit."""

    def get_commits_internal(commit_arg, repo_arg=None):
        """Get commit(s) by commit hash."""
        # Normally, this function should return only 1 commit object. However, if a repo and its forked repo
        # both appear in our workspace, there could be multiple commit objects.

        # If repo is not supplied, we need to scan all workspaces
        if repo_arg:
            _, vendor, repo_owner, repo_name = Gateway.parse_repo(repo_arg)
            workspaces = ['{}/{}/{}/{}'.format(Gateway.BASE_DIR, vendor, repo_owner, repo_name)]
        else:
            dirs = [dir[0] for dir in os.walk(Gateway.BASE_DIR) if dir[0].endswith('/.git')]
            workspaces = list(map(lambda v: v[:-5], dirs))

        commits = []
        with ThreadPoolExecutor(max_workers=20) as executor:
            futures = []
            for i in range(len(workspaces)):
                futures.append(executor.submit(Gateway._get_commit, workspaces[i], commit=commit_arg))
            for future in as_completed(futures):
                try:
                    data = future.result()
                    if data:
                        commits.append(data)
                except Exception as e:
                    logger.warning('Unexpected exception occurred during processing: %s', e)

        return commits

    repo = request.args.get('repo', "")
    if repo:
        repo = unquote(repo)
    logger.info('Retrieving commit (repo: %s, commit: %s) ...', repo, pk)
    commits_res = get_commits_internal(pk, repo)
    if not commits_res:
        logger.warning('Failed to retrieve commit')
        raise AXApiInvalidParam('Invalid revision', detail='Invalid revision ({})'.format(pk))
    else:
        if len(commits_res) > 1:
            logger.warning('Found multiple commits with given sha, returning the first one ...')
        logger.info('Successfully retrieved commit')
        return jsonify(commits_res[0])


@app.route('/v1/scm/files', methods=['PUT', 'DELETE'])
def files():
    """Get a single file content and upload to s3."""
    repo = request.args.get('repo')
    branch = request.args.get('branch')
    path = request.args.get('path')
    if not all([repo, branch, path]):
        raise AXApiInvalidParam('Missing required parameters', 'Missing required parameters (repo, branch, path)')
    if path.startswith('/'):
        path = path[1:]

    if request.method == 'PUT':
        resp = Gateway._put_file(repo, branch, path)
    else:
        resp = Gateway._delete_file(repo, branch, path)
    return jsonify(resp)


# API for Argo Approval
@app.route('/v1/results/<id>/approval', methods=['GET', ])
def approval(id):
    """Save an approval result in redis."""
    token = request.args.get('token', None)
    result = jwt.decode(token, 'ax', algorithms=['HS256'])
    redis_client = RedisClient('redis', db=DB_RESULT)
    result['timestamp'] = int(time.time())

    logger.info("Decode token {}, \n to {}".format(token, json.dumps(result, indent=2)))

    # differentiate key for approval result from the task result
    uuid = result['leaf_id'] + '-axapproval'
    try:
        logger.info("Setting approval result (%s) to Redis ...", uuid)
        try:
            state = gateway.axdb_client.get_approval_info(root_id=result['root_id'], leaf_id=result['leaf_id'])
            if state and state[0]['result'] != 'WAITING':
                return redirect("https://{}/error/404/type/ERR_AX_ILLEGAL_OPERATION;msg=The%20link%20is%20no%20longer%20valid.".format(result['dns']))

            if gateway.axdb_client.get_approval_results(leaf_id=result['leaf_id'], user=result['user']):
                return redirect("https://{}/error/404/type/ERR_AX_ILLEGAL_OPERATION;msg=Response%20has%20already%20been%20submitted.".format(result['dns']))

            # push result to redis (brpop)
            redis_client.rpush(uuid, value=result, encoder=json.dumps)
        except Exception as exc:
            logger.exception(exc)
            pass
        # save result to axdb
        gateway.axdb_client.create_approval_results(leaf_id=result['leaf_id'],
                                                    root_id=result['root_id'],
                                                    result=result['result'],
                                                    user=result['user'],
                                                    timestamp=result['timestamp'])
    except Exception as e:
        msg = 'Failed to save approval result to Redis: {}'.format(e)
        logger.error(msg)
        raise
    else:
        logger.info('Successfully saved result to Redis')
        return redirect("https://{}/success/201;msg=Response%20has%20been%20submitted%20successfully.".format(result['dns']))


# API for Nexus
@app.route('/v1/results/test_nexus_credential', methods=['PUT', ])
def test_nexus_credential():
    payload = get_json()
    logger.info('Received testing request (payload: %s)', payload)
    username = payload.get('username', "")
    password = payload.get('password', "")
    port = payload.get('port', 8081)
    hostname = payload.get('hostname', None)

    if not hostname:
        raise AXApiInvalidParam('Missing required parameters: Hostname', detail='Missing required parameters, hostname')

    response = requests.get('{}:{}/nexus/service/local/users'.format(hostname, port),
                            auth=(username, password), timeout=10)

    if response.ok:
        return jsonify({})
    else:
        response.raise_for_status()


# API for reporting platform notification
@app.route('/v1/results/redirect_notification_center', methods=['POST', ])
def redirect_notification_center():
    payload = get_json()
    logger.info('Received redirecting nc request (payload: %s)', payload)
    detail = payload.get('detail', "")
    try:
        gateway.event_notification_client.send_message_to_notification_center(
            CODE_PLATFORM_ERROR, detail={'message': detail})
    except Exception:
        logger.exception("Failed to send event to notification center")
        raise
    return jsonify({})


def _query_match(data, query_dict):
    for k, v in query_dict.items():
        if data.get(k, None) != v:
            return False
    return True


def _normalize_data(proj_dict):
    filtered_project_keys = ('id', 'key', 'name', 'projectTypeKey')
    return dict([(k, proj_dict.get(k, None)) for k in filtered_project_keys])


# APIs for JIRA
@app.route('/v1/jira/users', methods=['GET'])
def get_users():
    query_dict = dict()
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    filtered_users_keys = ('key', 'active', 'fullname', 'email')
    for pk in filtered_users_keys:
        pv = request.args.get(pk, None)
        if pk == 'active':
            if pv == 'true':
                pv = True
            elif pv == 'false':
                pv = False
        if pv is not None:
            query_dict[pk] = pv

    users = jira_client.users()
    users = [u for u in users if _query_match(u, query_dict)]
    return jsonify({'data': users})


@app.route('/v1/jira/projects', methods=['GET'])
def get_projects():
    query_dict = dict()
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    filtered_project_keys = ('id', 'key', 'name', 'projectTypeKey')
    for pk in filtered_project_keys:
        pv = request.args.get(pk, None)
        if pv is not None:
            query_dict[pk] = pv

    ps = jira_client.get_projects(json_result=True)
    ps = [p for p in ps if _query_match(p, query_dict)]
    ps = [_normalize_data(p) for p in ps]
    return jsonify({'data': ps})


@app.route('/v1/jira/projects/<name>', methods=['GET'])
def get_project(name):
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    proj = jira_client.get_project(name, json_result=True)
    return jsonify(_normalize_data(proj))


@app.route('/v1/jira/projects/test', methods=['POST'])
def test_project():
    payload = get_json()
    url = payload.get('url', '').lower()
    username = payload.get('username', None)
    password = payload.get('password', None)
    logger.info('Received request (url: %s, username: %s, password: ******)', url, username)

    assert all([url, username, password]), \
        AXApiInvalidParam('Missing required parameters', detail='Required parameters (username, password, url)')

    try:
        Gateway.init_jira_client(url, username, password)
    except requests.exceptions.ConnectionError as exc:
        raise AXApiInternalError('Invalid URL', detail=str(exc))
    except jira.exceptions.JIRAError as exc:
        raise AXApiInternalError('Invalid authentication', detail=str(exc))
    except Exception as exc:
        raise AXApiInternalError('Failed to connect to JIRA', detail=str(exc))
    else:
        return jsonify({})


@app.route('/v1/jira/issues', methods=['POST'])
def create_issue():
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    payload = get_json()
    logger.info('Received Jira issue creation request (%s)', payload)
    project = payload.get('project', None)
    summary = payload.get('summary', None)
    issuetype = payload.get('issuetype', None)
    reporter = payload.get('reporter', None)

    description = payload.get('description', None)  # optional

    if project is None:
        raise AXApiInvalidParam('Missing required parameters: Project',
                                detail='Missing required parameters, Project')
    if summary is None:
        raise AXApiInvalidParam('Missing required parameters: Summary',
                                detail='Missing required parameters, Summary')
    if issuetype is None:
        raise AXApiInvalidParam('Missing required parameters: Issuetype',
                                detail='Missing required parameters, Issuetype')
    if reporter is None:
        raise AXApiInvalidParam('Missing required parameters: Reporter',
                                detail='Missing required parameters, Reporter')

    try:
        issue_obj = jira_client.create_issue(project, summary,
                                             issuetype=issuetype, reporter=reporter, description=description)
    except jira.exceptions.JIRAError as exc:
        raise AXApiInternalError('Invalid Parameters', detail=str(exc))
    else:
        issue_dict = copy.deepcopy(issue_obj.raw['fields'])
        issue_dict['url'] = issue_obj.self
        issue_dict['id'] = issue_obj.id
        issue_dict['key'] = issue_obj.key
        return jsonify(issue_dict)


@app.route('/v1/jira/issues', methods=['GET'])
def get_issues():
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    filtered_issue_keys = ('project', 'status', 'component', 'labels', 'issuetype', 'priority',
                           'creator', 'assignee', 'reporter', 'fixversion', 'affectedversion')
    query_ids = request.args.get('ids', None)
    if query_ids is not None:
        issues = []
        with ThreadPoolExecutor(max_workers=5) as executor:
            futures = []
            id_list = query_ids.strip().split(',')
            logger.info('Query the following Jira issues: %s', id_list)
            for issue_id in id_list:
                futures.append(executor.submit(jira_client.get_issue, issue_id.strip(), json_result=True))
            for future in as_completed(futures):
                try:
                    issues.append(future.result())
                except Exception as exc:
                    logger.warning('Unexpected exception %s', exc)
    else:
        kwargs = dict()
        for key in request.query_params.keys():
            if key.lower() in filtered_issue_keys:
                kwargs[key.lower()] = request.query_params.get(key)
        logger.info('Query kwargs: %s:', kwargs)
        issues = jira_client.query_issues(json_result=True, **kwargs)

    return jsonify(issues)


@app.route('/v1/jira/issues/<issue_id>', methods=['GET'])
def get_issue(issue_id):
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    issue = jira_client.get_issue(issue_id, json_result=True)
    return jsonify(issue)


@app.route('/v1/jira/issues/<issue_id>/getcomments', methods=['GET'])
def get_comments(issue_id):
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    default_max_results = 3
    max_results = int(request.args.get('max_results', default_max_results))
    comments = jira_client.get_issue_comments(issue_id, latest_num=max_results, json_result=True)
    return jsonify(comments)


@app.route('/v1/jira/issues/<issue_id>/addcomments', methods=['POST'])
def add_comment(issue_id):
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    payload = get_json()
    comment = payload.get('comment', None)
    user = payload.get('user', None)

    if not comment:
        raise AXApiInvalidParam('Require Comment message info')
    if not user:
        raise AXApiInvalidParam('Require Commenter info')

    try:
        jira_client.add_issue_comment(issue_id, comment, commenter=user)
    except Exception as exc:
        raise AXApiInternalError('Failed to add comment', detail=str(exc))
    return jsonify({})


@app.route('/v1/jira/issuetypes', methods=['GET'])
def get_issue_types():
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    issue_types = jira_client.get_issue_types(json_result=True)
    return jsonify({'data': issue_types})


@app.route('/v1/jira/issuetypes/<issue_type>', methods=['GET'])
def get_issue_type(issue_type):
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    issue_type = jira_client.get_issue_type_by_name(issue_type, json_result=True)
    return jsonify(issue_type)


@app.route('/v1/jira/webhooks', methods=['GET'])
def get_webhooks():
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    ax_webhooks = jira_client.get_ax_webhooks()
    return jsonify({'data': ax_webhooks})


@app.route('/v1/jira/webhooks', methods=['POST'])
def create_webhook():
    payload = get_json()
    logger.info('Received jira webhook creation request, %s', payload)

    url = payload.get('url', None)
    username = payload.get('username', None)
    password = payload.get('password', None)
    webhook = payload.get('webhook', None)
    projects = payload.get('projects', None)

    # Create ingress
    try:
        dnsname = urlparse(webhook).netloc
        logger.info('Creating ingress for Jira webhook %s', dnsname)
        gateway.axsys_client.create_ingress(dnsname)
    except Exception as exc:
        logger.error('Failed to create ingress for webhook: %s', str(exc))
        raise AXApiInternalError('Failed to create ingress for webhook', str(exc))
    else:
        logger.info('Successfully created ingress for webhook')

    # Create webhook
    jira_client = Gateway.init_jira_client(gateway.axops_client, url=url, username=username, password=password)
    try:
        if projects:
            logger.info('Filtered projects are: %s', projects)
            if type(projects) == str:
                projects = json.loads(projects)
        else:
            logger.info('No project filter')
            projects = None
        wh = jira_client.create_ax_webhook(webhook, projects=projects)
    except Exception as exc:
        logger.exception(exc)
        raise AXApiInternalError('Fail to create Jira webhooks', detail=str(exc))
    return jsonify(wh.json())


@app.route('/v1/jira/webhooks', methods=['PUT'])
def modify_webhook():
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    payload = get_json()
    projects = payload.get('projects', None)
    logger.info('Received jira webhook update request, %s', payload)
    # Update webhook
    try:
        if projects:
            logger.info('Filtered projects are: %s', projects)
            if type(projects) == str:
                projects = json.loads(projects)
        else:
            logger.info('No project filter')
            projects = None
        jira_client.update_ax_webhook(projects)
    except Exception as exc:
        logger.exception(exc)
        raise AXApiInternalError('Fail to update jira webhooks', detail=str(exc))
    else:
        logger.info('Successfully updated Jira webhook')
    return jsonify({})


@app.route('/v1/jira/webhooks', methods=['DELETE'])
def delete_webhook():
    jira_client = Gateway.init_jira_client(gateway.axops_client)
    wh = jira_client.get_ax_webhook()
    if not wh:
        logger.warning('No webhook on Jira server, ignore it')
        return jsonify({})

    # Delete ingress
    try:
        logger.info('Deleting ingress for Jira webhook %s', wh['url'])
        gateway.axsys_client.delete_ingress(urlparse(wh['url']).netloc)
    except Exception as exc:
        logger.error('Failed to delete ingress for webhook: %s', str(exc))
        raise AXApiInternalError('Failed to delete ingress for webhook', str(exc))
    else:
        logger.info('Successfully deleted ingress for webhook')
    # Delete webhook
    try:
        jira_client.delete_ax_webhook()
    except Exception as exc:
        logger.exception(exc)
        raise AXApiInternalError('Fail to delete jira webhooks', detail=str(exc))
    return jsonify({})


@app.route('/v1/jira/events$', methods=['POST'])
def create_jira_event():
    checked_fields = ('description', 'project', 'status', 'summary', 'Key')
    delete_event = 'jira:issue_deleted'
    update_event = 'jira:issue_updated'

    payload = get_json()
    try:
        logger.info('Translating JIRA event ...')
        event = translate_jira_issue_event(payload)
    except Exception as exc:
        logger.error('Failed to translate event: %s', exc)
        raise AXApiInternalError('Failed to translate event', detail=str(exc))
    else:
        logger.info('Successfully translated event: %s', event)

    try:
        if event['type'] == delete_event:
            logger.info('The following Jira field(s) get updated: %s', event['changed_fields'])
            if event['status_category_id'] == 3:
                logger.info('Jira issue %s is closed', event['id'])
                logger.info('Delete Jira on AXDB %s', event['id'])
                gateway.axops_client.delete_jira_issue(event['id'])
            elif event['changed_fields'] and any(f in event['changed_fields'] for f in checked_fields):
                logger.info('Update Jira content on AXDB ...')
                gateway.axops_client.update_jira_issue(event['axdb_content'])
            else:
                logger.info('No Jira content need to be updated')
        elif event['type'] == update_event:
            logger.info('Delete Jira on AXDB %s', event['id'])
            gateway.axops_client.delete_jira_issue(event['id'])
        else:
            logger.warning('Not supported event: (%s), ignore it', event['type'])
    except Exception as exc:
        raise AXApiInternalError('Failed to update JIRA content on AXDB', detail=str(exc))
    else:
        return jsonify({})

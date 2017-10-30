import * as moment from 'moment';
import { Injectable, NgZone } from '@angular/core';
import { Observable, Observer, Subscription } from 'rxjs';
import { Http, Headers, URLSearchParams } from '@angular/http';
import { HttpService } from './http.service';
import { ViewPreferencesService } from './view-preferences.service';
import { AxHeaders } from './headers';
import { DEFAULT_NOTIFICATIONS } from '../model';

import { Task, TaskFieldNames, TaskStatus, TaskCreationArgs, Branch, BranchTasks } from '../model';

export interface ServiceEvent { task_id: string; id: string; status: TaskStatus; repo: string; branch: string; }

@Injectable()
export class TaskService {

    constructor(private http: Http, private httpService: HttpService, private zone: NgZone, private viewPreferencesService: ViewPreferencesService) {
    }

    public getTask(id: string, noLoader = false, noErrorHandling = false): Observable<Task> {
        return this.http.get(`v1/services/${id}`, { headers: new AxHeaders({ noLoader, noErrorHandling }) })
            .map(res => <Task>res.json());
    }

    public getTemplateHistories(templateId: string, commitId: string) {
        return this.http.get(`v1/services?template=${templateId}&revision=${commitId}`)
            .map(res => res.json());
    }

    public cancelTask(id: string) {
        return this.http.delete(`v1/services/${id}`);
    }

    public launchTask(args: TaskCreationArgs, runPartial = false) {
        if (!args.hasOwnProperty('notifications') && !args.notifications) {
            args = Object.assign({}, args, { notifications: DEFAULT_NOTIFICATIONS });
        }
        return this.http.post(`v1/services?run_partial=${runPartial}`, JSON.stringify(args))
            .do(res => {
                this.viewPreferencesService.updateViewPreferences(preferences => {
                    if (!preferences.firstJobFeedbackStatus) {
                        preferences.firstJobFeedbackStatus = 'need-feedback';
                    }
                });
            })
            .map(res => res.json());
    }

    public getTasks(params?: {
        startTime?: moment.Moment,
        endTime?: moment.Moment,
        limit?: number,
        offset?: number,
        branch?: string,
        repo?: string,
        commit?: string,
        revision?: string,
        template_name?: string,
        templateIds?: string,
        labels?: string,
        fields?: string[],
        searchFields?: string[],
        includeDetails?: boolean,
        branches?: Branch[],
        isActive?: boolean,
        username?: string[],
        status?: string[],
        tags?: string[],
        search?: string,
    }, isUpdated?: boolean): Observable<{ data: Task[] }> {
        let customHeader = new Headers(), labelQueryFrag = '';
        if (isUpdated) {
            customHeader.append('isUpdated', isUpdated.toString());
        }
        params = params || {};
        let search = new URLSearchParams();
        search.set('task_only', 'true');
        if (params.limit) {
            search.set('limit', params.limit.toString());
        }
        if (params.username) {
            search.set('username', params.username.join(','));
        }
        if (params.offset) {
            search.set('offset', params.offset.toString());
        }
        if (params.startTime) {
            search.set('min_time', params.startTime.unix().toString());
        }
        if (params.endTime) {
            search.set('max_time', params.endTime.unix().toString());
        }
        if (params.commit) {
            search.set('commit', params.commit);
        }
        if (params.revision) {
            search.set('revision', params.revision);
        }
        if (params.template_name) {
            search.set('template_name', params.template_name);
        }
        if (params.repo) {
            search.set('repo', params.repo);
        }
        if (params.branch) {
            search.set('branch', params.branch);
        }
        if (params.branches && params.branches.length > 0) {
            // If you are filtering by single repo and branch, please use repo=XXX&branch=XXX, then you will have more deterministic result.
            if (params.branches.length === 1) {
                search.set('repo', params.branches[0].repo);
                search.set('branch', params.branches[0].name);
            } else {
                search.set('repo_branch', JSON.stringify(params.branches.map(item => {
                    return { repo: item.repo, branch: item.name };
                })));
            }
        }
        if (params.templateIds) {
            search.set('template_id', params.templateIds);
        }
        if (params.includeDetails) {
            search.set('include_details', params.includeDetails.toString());
        }
        if (params.hasOwnProperty('isActive')) {
            search.set('is_active', params.isActive.toString());
        }
        if (params.labels) {
            // special treatment for labels as angular misbehaves with qparams
            // it does not encode ; as it expects to work with matrix urls
            // Note: backend api only expects ';' to be encoded not the ':'
            labelQueryFrag = `labels=${params.labels}`;
        }
        if (params.fields) {
            search.set('fields', params.fields.join(','));
        }

        if (params.searchFields) {
            search.set('search_fields', params.searchFields.join(','));
        }

        if (params.status) {
            search.set('status', params.status.join(','));
        }

        if (params.tags) {
            search.set('tags', params.tags.join(','));
        }

        if (params.search) {
            search.set('search', params.search.toString());
        }

        let query = labelQueryFrag ? labelQueryFrag + '&' + search.toString() :
            search.toString();

        return this.http.get(`v1/services?${query}`, { headers: customHeader }).map(res => <{ data: Task[] }>res.json());
    }

    public getTasksByBranches(params: {
        startTime?: moment.Moment,
        endTime?: moment.Moment,
        branch?: string,
        repo?: string,
        branches?: Branch[],
        isActive?: boolean
    }, hideLoader?: boolean): Observable<BranchTasks[]> {
        let headers = new Headers();
        let search = new URLSearchParams();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        search.set('task_only', 'true');
        if (params.startTime) {
            search.set('min_time', params.startTime.unix().toString());
        }
        if (params.endTime) {
            search.set('max_time', params.endTime.unix().toString());
        }

        if (params.repo && params.branch) {
            search.set('repo_branch', JSON.stringify([{ repo: params.repo, branch: params.branch }]));
        } else if (params.repo) {
            search.set('repo', params.repo);
        } else if (params.branch) {
            search.set('branch', params.branch);
        }
        if (params.branches && params.branches.length > 0) {
            search.set('repo_branch', JSON.stringify(params.branches.map(item => {
                return { repo: item.repo, branch: item.name };
            })));
        }
        if (params.hasOwnProperty('isActive')) {
            search.set('is_active', params.isActive.toString());
        }
        search.set('fields', [TaskFieldNames.status, TaskFieldNames.commit, TaskFieldNames.branch, TaskFieldNames.repo].join(','));
        return this.http
            .get(`v1/services`, { headers, search }).map(res => res.json())
            .map((res: { data: Task[] }) => {
                let branchByTasks = new Map<string, Task[]>();
                res.data.forEach(task => {
                    let key = `${task.branch}:${task.repo}`;
                    let tasks = branchByTasks.get(key) || [];
                    tasks.push(task);
                    branchByTasks.set(key, tasks);
                });
                return Array.from<[string, Task[]]>(branchByTasks.entries()).map(entry => {
                    let [key, tasks] = entry;
                    let index = key.indexOf(':');
                    let [branch, repo] = [key.slice(0, index), key.slice(index + 1)];
                    return {
                        branch: branch,
                        repo: repo,
                        tasks: tasks
                    };
                });
            });
    }

    public getTaskLogs(id: string): Observable<string> {
        return this.httpService.loadEventSource(`v1/services/${id}/logs`).map(data => JSON.parse(data).log);
    }

    public getTaskStepEvents(id: string): Observable<ServiceEvent> {
        return this.httpService.loadEventSource(`v1/services/${id}/events`).map(data => JSON.parse(data));
    }

    public getTasksEvents(repo?: string, branch?: string): Observable<ServiceEvent> {
        return Observable.create((observer: Observer<ServiceEvent>) => {
            let done = false;
            let subscription: Subscription = null;
            function ensureUnsubscribed() {
                if (subscription) {
                    subscription.unsubscribe();
                    subscription = null;
                }
            }

            let subscribeToEvents = () => {
                let url = 'v1/service/events';
                if (repo || branch) {
                    url += `?repo_branch=${JSON.stringify([{ repo, branch }])}`;
                }

                subscription = this.httpService.loadEventSource(url).map(data => JSON.parse(data)).subscribe((serviceEvent: ServiceEvent) => {
                    observer.next(serviceEvent);
                }, err => subscribeToEvents(), () => subscribeToEvents());
            };

            subscribeToEvents();
            return () => {
                done = true;
                ensureUnsubscribed();
            };
        });
    }

    /**
     * Returns observable which emits updated task info every time when task status got changed.
     */
    public getTaskUpdates(id: string, hideLoader = false): Observable<Task> {
        return Observable.create((observer: Observer<Task>) => {
            let subscription: Subscription = null;

            let ensureUnsubscribed = () => {
                if (subscription != null) {
                    subscription.unsubscribe();
                    subscription = null;
                }
            };

            let doLoadUpdates = (subscribeToEvents) => {
                this.getTask(id, hideLoader).toPromise().then(task => {
                    this.zone.run(() => observer.next(task));
                    hideLoader = true;
                    if (subscribeToEvents && [TaskStatus.Init, TaskStatus.Waiting, TaskStatus.Running].indexOf(task.status) > -1) {
                        ensureUnsubscribed();
                        subscription = this.getTaskStepEvents(task.id).subscribe(
                            () => doLoadUpdates(false), () => doLoadUpdates(true), () => doLoadUpdates(true));
                    }
                });
            };

            doLoadUpdates(true);

            return ensureUnsubscribed;
        });
    }

    public getTasksForFixture(id: string, fields: string[]): Promise<Task[]> {
        let search = new URLSearchParams();
        search.set('fixtures', id);
        if (fields.length > 0) {
            search.set('fields', fields.join(','));
        }
        return this.http.get('v1/services', { search , headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json().data);
    }

    /**
     * Creates WebSocket which executes given command in service container and allows sending message into app stdin and receiving
     * app stdout data.
     */
    public connectToConsole(uri: string, params: URLSearchParams) {
        let search = params || new URLSearchParams();
        let scheme = location.protocol === 'http:' ? 'ws' : 'wss';
        let socket = new WebSocket(`${scheme}://${location.hostname}:${location.port}/${uri}?${search.toString()}`);
        socket.binaryType = 'arraybuffer';
        return socket;
    }
}

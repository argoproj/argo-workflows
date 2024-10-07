import {Page} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {PaginationPanel} from '../../../shared/components/pagination-panel';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {formatDuration, wfDuration} from '../../../shared/duration';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {ScopedLocalStorage} from '../../../shared/scoped-local-storage';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {ArchivedWorkflowFilters} from '../archived-workflow-filters/archived-workflow-filters';

interface BrowserStorageOptions {
    pagination: Pagination;
    namespace: string;
    name: string;
    namePrefix: string;
    selectedPhases: string[];
    selectedLabels: string[];
    minStartedAt?: Date;
    maxStartedAt?: Date;
    error?: Error;
    deep: boolean;
}

interface State extends BrowserStorageOptions {
    workflows?: Workflow[];
}

const defaultPaginationLimit = 10;

export class ArchivedWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private storage: ScopedLocalStorage;

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.storage = new ScopedLocalStorage('ArchiveListOptions');
        const savedOptions = this.storage.getItem('options', {
            pagination: {limit: defaultPaginationLimit},
            selectedPhases: [],
            selectedLabels: []
        } as State);
        const phaseQueryParam = this.queryParams('phase');
        const labelQueryParam = this.queryParams('label');
        this.state = {
            pagination: {offset: this.queryParam('offset'), limit: parseLimit(this.queryParam('limit')) || savedOptions.pagination.limit},
            namespace: Utils.getNamespace(this.props.match.params.namespace) || '',
            name: this.queryParams('name').toString() || '',
            namePrefix: this.queryParams('namePrefix').toString() || '',
            selectedPhases: phaseQueryParam.length > 0 ? phaseQueryParam : savedOptions.selectedPhases,
            selectedLabels: labelQueryParam.length > 0 ? labelQueryParam : savedOptions.selectedLabels,
            minStartedAt: this.parseTime(this.queryParam('minStartedAt')) || this.lastMonth(),
            maxStartedAt: this.parseTime(this.queryParam('maxStartedAt')) || this.nextDay(),
            deep: this.queryParam('deep') === 'true'
        };
    }

    public componentDidMount(): void {
        this.fetchArchivedWorkflows(
            this.state.namespace,
            this.state.name,
            this.state.namePrefix,
            this.state.selectedPhases,
            this.state.selectedLabels,
            this.state.minStartedAt,
            this.state.maxStartedAt,
            this.state.pagination
        );
        services.info.collectEvent('openedArchivedWorkflowList').then();
    }

    public componentDidUpdate(): void {
        if (this.state.deep === true && this.state.workflows && this.state.workflows.length === 1) {
            const workflow = this.state.workflows[0];
            const url = uiUrl('archived-workflows/' + workflow.metadata.namespace + '/' + (workflow.metadata.uid || ''));
            this.props.history.push(url);
        }
    }

    public render() {
        return (
            <Page
                title='Archived Workflows'
                toolbar={{
                    breadcrumbs: [
                        {title: 'Archived Workflows', path: uiUrl('archived-workflows')},
                        {title: this.state.namespace, path: uiUrl('archived-workflows/' + this.state.namespace)}
                    ]
                }}>
                <div className='row'>
                    <div className='columns small-12 xlarge-2'>
                        <div>
                            <ArchivedWorkflowFilters
                                workflows={this.state.workflows || []}
                                namespace={this.state.namespace}
                                name={this.state.name}
                                namePrefix={this.state.namePrefix}
                                phaseItems={Object.values([models.NODE_PHASE.SUCCEEDED, models.NODE_PHASE.FAILED, models.NODE_PHASE.ERROR])}
                                selectedPhases={this.state.selectedPhases}
                                selectedLabels={this.state.selectedLabels}
                                minStartedAt={this.state.minStartedAt}
                                maxStartedAt={this.state.maxStartedAt}
                                onChange={(namespace, name, namePrefix, selectedPhases, selectedLabels, minStartedAt, maxStartedAt) =>
                                    this.changeFilters(namespace, name, namePrefix, selectedPhases, selectedLabels, minStartedAt, maxStartedAt, {
                                        limit: this.state.pagination.limit
                                    })
                                }
                            />
                        </div>
                    </div>
                    <div className='columns small-12 xlarge-10'>{this.renderWorkflows()}</div>
                </div>
            </Page>
        );
    }

    private lastMonth() {
        const dt = new Date();
        dt.setMonth(dt.getMonth() - 1);
        dt.setHours(0, 0, 0, 0);
        return dt;
    }

    private nextDay() {
        const dt = new Date();
        dt.setDate(dt.getDate() + 1);
        dt.setHours(0, 0, 0, 0);
        return dt;
    }

    private parseTime(dateStr: string) {
        if (dateStr != null) {
            return new Date(dateStr);
        }
    }

    private changeFilters(
        namespace: string,
        name: string,
        namePrefix: string,
        selectedPhases: string[],
        selectedLabels: string[],
        minStartedAt: Date,
        maxStartedAt: Date,
        pagination: Pagination
    ) {
        this.fetchArchivedWorkflows(namespace, name, namePrefix, selectedPhases, selectedLabels, minStartedAt, maxStartedAt, pagination);
    }

    private get filterParams() {
        const params = new URLSearchParams();
        if (this.state.selectedPhases) {
            this.state.selectedPhases.forEach(phase => {
                params.append('phase', phase);
            });
        }
        if (this.state.selectedLabels) {
            this.state.selectedLabels.forEach(label => {
                params.append('label', label);
            });
        }
        if (this.state.name) {
            params.append('name', this.state.name);
        }
        if (this.state.namePrefix) {
            params.append('namePrefix', this.state.namePrefix);
        }
        params.append('minStartedAt', this.state.minStartedAt.toISOString());
        params.append('maxStartedAt', this.state.maxStartedAt.toISOString());
        if (this.state.pagination.offset) {
            params.append('offset', this.state.pagination.offset);
        }
        if (this.state.pagination.limit !== null && this.state.pagination.limit !== defaultPaginationLimit) {
            params.append('limit', this.state.pagination.limit.toString());
        }
        return params;
    }

    private fetchBrowserStorageStateObject(state: State): BrowserStorageOptions {
        const browserStorageOptions: BrowserStorageOptions = {} as BrowserStorageOptions;
        browserStorageOptions.deep = state.deep;
        browserStorageOptions.error = state.error;
        browserStorageOptions.maxStartedAt = state.maxStartedAt;
        browserStorageOptions.minStartedAt = state.minStartedAt;
        browserStorageOptions.name = state.name;
        browserStorageOptions.namePrefix = state.namePrefix;
        browserStorageOptions.namespace = state.namespace;
        browserStorageOptions.pagination = state.pagination;
        browserStorageOptions.selectedLabels = state.selectedLabels;
        browserStorageOptions.selectedPhases = state.selectedPhases;
        return browserStorageOptions;
    }

    private saveHistory() {
        this.storage.setItem('options', this.fetchBrowserStorageStateObject(this.state), {} as BrowserStorageOptions);
        const newNamespace = Utils.managedNamespace ? '' : this.state.namespace;
        this.url = uiUrl('archived-workflows' + (newNamespace ? '/' + newNamespace : '') + '?' + this.filterParams.toString());
        Utils.currentNamespace = this.state.namespace;
    }

    private fetchArchivedWorkflows(
        namespace: string,
        name: string,
        namePrefix: string,
        selectedPhases: string[],
        selectedLabels: string[],
        minStartedAt: Date,
        maxStartedAt: Date,
        pagination: Pagination
    ): void {
        services.archivedWorkflows
            .list(namespace, name, namePrefix, selectedPhases, selectedLabels, minStartedAt, maxStartedAt, pagination)
            .then(list => {
                this.setState(
                    {
                        error: null,
                        namespace,
                        name,
                        namePrefix,
                        workflows: list.items || [],
                        selectedPhases,
                        selectedLabels,
                        minStartedAt,
                        maxStartedAt,
                        pagination: {
                            limit: pagination.limit,
                            offset: pagination.offset,
                            nextOffset: list.metadata.continue
                        }
                    },
                    this.saveHistory
                );
            })
            .catch(error => this.setState({error}));
    }

    private renderWorkflows() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.workflows) {
            return <Loading />;
        }
        const learnMore = <a href='https://argo-workflows.readthedocs.io/en/release-3.4/workflow-archive/'>Learn more</a>;
        if (this.state.workflows.length === 0) {
            return (
                <ZeroState title='No archived workflows'>
                    <p>To add entries to the archive you must enable archiving in configuration. Records are created in the archive on workflow completion.</p>
                    <p>{learnMore}.</p>
                </ZeroState>
            );
        }

        return (
            <>
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1' />
                        <div className='columns small-3'>NAME</div>
                        <div className='columns small-2'>NAMESPACE</div>
                        <div className='columns small-2'>STARTED</div>
                        <div className='columns small-2'>FINISHED</div>
                        <div className='columns small-2'>DURATION</div>
                    </div>
                    {this.state.workflows.map(w => (
                        <Link className='row argo-table-list__row' key={`${w.metadata.uid}`} to={uiUrl(`archived-workflows/${w.metadata.namespace}/${w.metadata.uid}`)}>
                            <div className='columns small-1'>
                                <PhaseIcon value={w.status.phase} />
                            </div>
                            <div className='columns small-3'>{w.metadata.name}</div>
                            <div className='columns small-2'>{w.metadata.namespace}</div>
                            <div className='columns small-2'>
                                <Timestamp date={w.status.startedAt} />
                            </div>
                            <div className='columns small-2'>
                                <Timestamp date={w.status.finishedAt} />
                            </div>
                            <div className='columns small-2'>{formatDuration(wfDuration(w.status))}</div>
                        </Link>
                    ))}
                </div>
                <PaginationPanel
                    onChange={pagination =>
                        this.changeFilters(
                            this.state.namespace,
                            this.state.name,
                            this.state.namePrefix,
                            this.state.selectedPhases,
                            this.state.selectedLabels,
                            this.state.minStartedAt,
                            this.state.maxStartedAt,
                            pagination
                        )
                    }
                    pagination={this.state.pagination}
                    numRecords={(this.state.workflows || []).length}
                />
                <p>
                    <i className='fa fa-info-circle' /> Records are created in the archive when a workflow completes. {learnMore}.
                </p>
            </>
        );
    }
}

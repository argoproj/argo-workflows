import {Page} from 'argo-ui';

import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {PaginationPanel} from '../../../shared/components/pagination-panel';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {formatDuration, wfDuration} from '../../../shared/duration';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {ArchivedWorkflowFilters} from '../archived-workflow-filters/archived-workflow-filters';

interface State {
    pagination: Pagination;
    loading: boolean;
    initialized: boolean;
    managedNamespace: boolean;
    namespace: string;
    selectedPhases: string[];
    selectedLabels: string[];
    minStartedAt?: Date;
    maxStartedAt?: Date;
    workflows?: Workflow[];
    error?: Error;
}

const defaultPaginationLimit = 10;

export class ArchivedWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {
            loading: true,
            pagination: {offset: this.queryParam('offset'), limit: parseLimit(this.queryParam('limit')) || defaultPaginationLimit},
            initialized: false,
            managedNamespace: false,
            namespace: this.props.match.params.namespace || Utils.getCurrentNamespace() || '',
            selectedPhases: this.queryParams('phase'),
            selectedLabels: this.queryParams('label'),
            minStartedAt: this.parseTime(this.queryParam('minStartedAt')) || this.lastMonth(),
            maxStartedAt: this.parseTime(this.queryParam('maxStartedAt')) || this.nextDay()
        };
    }

    public componentDidMount(): void {
        this.fetchArchivedWorkflows(
            this.state.namespace,
            this.state.selectedPhases,
            this.state.selectedLabels,
            this.state.minStartedAt,
            this.state.maxStartedAt,
            this.state.pagination
        );
    }

    public render() {
        if (this.state.loading) {
            return <Loading />;
        }
        if (this.state.error) {
            throw this.state.error;
        }
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Archived Workflows'
                        toolbar={{
                            breadcrumbs: [{title: 'Archived Workflows', path: uiUrl('archived-workflows')}]
                        }}>
                        <div className='row'>
                            <div className='columns small-12 xlarge-2'>
                                <div>
                                    <ArchivedWorkflowFilters
                                        workflows={this.state.workflows}
                                        namespace={this.state.namespace}
                                        phaseItems={Object.values([models.NODE_PHASE.SUCCEEDED, models.NODE_PHASE.FAILED, models.NODE_PHASE.ERROR])}
                                        selectedPhases={this.state.selectedPhases}
                                        selectedLabels={this.state.selectedLabels}
                                        minStartedAt={this.state.minStartedAt}
                                        maxStartedAt={this.state.maxStartedAt}
                                        onChange={(namespace, selectedPhases, selectedLabels, minStartedAt, maxStartedAt) =>
                                            this.changeFilters(namespace, selectedPhases, selectedLabels, minStartedAt, maxStartedAt, {limit: this.state.pagination.limit})
                                        }
                                    />
                                </div>
                            </div>
                            <div className='columns small-12 xlarge-10'>{this.renderWorkflows()}</div>
                        </div>
                    </Page>
                )}
            </Consumer>
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

    private changeFilters(namespace: string, selectedPhases: string[], selectedLabels: string[], minStartedAt: Date, maxStartedAt: Date, pagination: Pagination) {
        const params = new URLSearchParams();
        selectedPhases.forEach(phase => {
            params.append('phase', phase);
        });
        selectedLabels.forEach(label => {
            params.append('label', label);
        });
        params.append('minStartedAt', minStartedAt.toISOString());
        params.append('maxStartedAt', maxStartedAt.toISOString());
        if (pagination.offset) {
            params.append('offset', pagination.offset);
        }
        if (pagination.limit !== defaultPaginationLimit) {
            params.append('limit', pagination.limit.toString());
        }
        const url = 'archived-workflows/' + namespace + '?' + params.toString();
        history.pushState(null, '', uiUrl(url));
        this.fetchArchivedWorkflows(namespace, selectedPhases, selectedLabels, minStartedAt, maxStartedAt, pagination);
    }

    private fetchArchivedWorkflows(namespace: string, selectedPhases: string[], selectedLabels: string[], minStartedAt: Date, maxStartedAt: Date, pagination: Pagination): void {
        let archivedWorkflowList;
        let newNamespace = namespace;
        if (!this.state.initialized) {
            archivedWorkflowList = services.info.getInfo().then(info => {
                if (info.managedNamespace) {
                    newNamespace = info.managedNamespace;
                }
                this.setState({initialized: true, managedNamespace: !!info.managedNamespace});
                return services.archivedWorkflows.list(newNamespace, selectedPhases, selectedLabels, minStartedAt, maxStartedAt, pagination);
            });
        } else {
            if (this.state.managedNamespace) {
                newNamespace = this.state.namespace;
            }
            archivedWorkflowList = services.archivedWorkflows.list(newNamespace, selectedPhases, selectedLabels, minStartedAt, maxStartedAt, pagination);
        }
        archivedWorkflowList
            .then(list => {
                this.setState({
                    namespace: newNamespace,
                    workflows: list.items || [],
                    selectedPhases,
                    selectedLabels,
                    minStartedAt,
                    maxStartedAt,
                    pagination: {
                        limit: pagination.limit,
                        offset: pagination.offset,
                        nextOffset: list.metadata.continue
                    },
                    loading: false
                });
                Utils.setCurrentNamespace(newNamespace);
            })
            .catch(error => this.setState({error, loading: false}));
    }

    private renderWorkflows() {
        if (!this.state.workflows) {
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/master/docs/workflow-archive.md'>Learn more</a>;
        if (this.state.workflows.length === 0) {
            return (
                <ZeroState title='No archived workflows'>
                    <p>To add entries to the archive you must enabled archiving in configuration. Records are the created in the archive on workflow completion.</p>
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
                        this.changeFilters(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, this.state.minStartedAt, this.state.maxStartedAt, pagination)
                    }
                    pagination={this.state.pagination}
                />
                <p>
                    <i className='fa fa-info-circle' /> Records are created in the archive when a workflow completes. {learnMore}.
                </p>
            </>
        );
    }
}

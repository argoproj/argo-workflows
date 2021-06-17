import {Autocomplete, Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {labels, Workflow, WorkflowPhase, WorkflowPhases} from '../../../../models';
import {uiUrl} from '../../../shared/base';

import {BasePage} from '../../../shared/components/base-page';

import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {Loading} from '../../../shared/components/loading';
import {PaginationPanel} from '../../../shared/components/pagination-panel';
import {Query} from '../../../shared/components/query';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {ListWatch, sortByYouth} from '../../../shared/list-watch';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {ScopedLocalStorage} from '../../../shared/scoped-local-storage';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import * as Actions from '../../../shared/workflow-operations-map';
import {WorkflowCreator} from '../workflow-creator';
import {WorkflowFilters} from '../workflow-filters/workflow-filters';
import {WorkflowsRow} from '../workflows-row/workflows-row';
import {WorkflowsToolbar} from '../workflows-toolbar/workflows-toolbar';

require('./workflows-list.scss');

interface State {
    namespace: string;
    pagination: Pagination;
    selectedPhases: WorkflowPhase[];
    selectedLabels: string[];
    selectedWorkflows: Map<string, models.Workflow>;
    workflows?: Workflow[];
    resourceVersion?: string;
    error?: Error;
    batchActionDisabled: Actions.OperationDisabled;
}

interface WorkflowListRenderOptions {
    paginationLimit: number;
    selectedPhases: WorkflowPhase[];
    selectedLabels: string[];
}

const allBatchActionsEnabled: Actions.OperationDisabled = {
    RETRY: false,
    RESUBMIT: false,
    SUSPEND: false,
    RESUME: false,
    STOP: false,
    TERMINATE: false,
    DELETE: false
};

export class WorkflowsList extends BasePage<RouteComponentProps<any>, State> {
    private storage: ScopedLocalStorage;

    private get sidePanel() {
        return this.queryParam('sidePanel');
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
        if (this.state.pagination.offset) {
            params.append('offset', this.state.pagination.offset);
        }
        if (this.state.pagination.limit) {
            params.append('limit', this.state.pagination.limit.toString());
        }
        return params;
    }

    private get options() {
        const options: WorkflowListRenderOptions = {} as WorkflowListRenderOptions;
        options.selectedPhases = this.state.selectedPhases;
        options.selectedLabels = this.state.selectedLabels;
        if (this.state.pagination.limit) {
            options.paginationLimit = this.state.pagination.limit;
        }
        return options;
    }

    private listWatch: ListWatch<Workflow>;

    constructor(props: RouteComponentProps<State>, context: any) {
        super(props, context);
        this.storage = new ScopedLocalStorage('ListOptions');
        const savedOptions = this.storage.getItem('options', {
            paginationLimit: 0,
            selectedPhases: [],
            selectedLabels: []
        } as WorkflowListRenderOptions);
        const phaseQueryParam = this.queryParams('phase');
        const labelQueryParam = this.queryParams('label');
        this.state = {
            pagination: {
                offset: this.queryParam('offset'),
                limit: parseLimit(this.queryParam('limit')) || savedOptions.paginationLimit || 500
            },
            namespace: Utils.getNamespace(this.props.match.params.namespace) || '',
            selectedPhases: phaseQueryParam.length > 0 ? (phaseQueryParam as WorkflowPhase[]) : savedOptions.selectedPhases,
            selectedLabels: labelQueryParam.length > 0 ? (labelQueryParam as string[]) : savedOptions.selectedLabels,
            selectedWorkflows: new Map<string, models.Workflow>(),
            batchActionDisabled: {...allBatchActionsEnabled}
        };
    }

    public componentDidMount(): void {
        this.setState({selectedWorkflows: new Map<string, models.Workflow>()}, () => {
            this.fetchWorkflows(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, this.state.pagination);
        });
    }

    public componentWillUnmount(): void {
        this.setState({selectedWorkflows: new Map<string, models.Workflow>()});
        if (this.listWatch) {
            this.listWatch.stop();
        }
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflows'
                        toolbar={{
                            breadcrumbs: [
                                {title: 'Workflows', path: uiUrl('workflows')},
                                {title: this.state.namespace, path: uiUrl('workflows/' + this.state.namespace)}
                            ],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Submit New Workflow',
                                        iconClassName: 'fa fa-plus',
                                        action: () => ctx.navigation.goto('.', {sidePanel: 'submit-new-workflow'})
                                    }
                                ]
                            }
                        }}>
                        <WorkflowsToolbar
                            selectedWorkflows={this.state.selectedWorkflows}
                            clearSelection={() => this.setState({selectedWorkflows: new Map<string, models.Workflow>()})}
                            loadWorkflows={() => {
                                this.setState({selectedWorkflows: new Map<string, models.Workflow>()});
                                this.changeFilters(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, {limit: this.state.pagination.limit});
                            }}
                            isDisabled={this.state.batchActionDisabled}
                        />
                        <div className='row'>
                            <div className='columns small-12 xlarge-2'>
                                {this.renderQuery(ctx)}
                                <div>
                                    <WorkflowFilters
                                        workflows={this.state.workflows || []}
                                        namespace={this.state.namespace}
                                        phaseItems={WorkflowPhases}
                                        selectedPhases={this.state.selectedPhases}
                                        selectedLabels={this.state.selectedLabels}
                                        onChange={(namespace, selectedPhases, selectedLabels) =>
                                            this.changeFilters(namespace, selectedPhases, selectedLabels, {limit: this.state.pagination.limit})
                                        }
                                    />
                                </div>
                            </div>
                            <div className='columns small-12 xlarge-10'>{this.renderWorkflows()}</div>
                        </div>
                        <SlidingPanel isShown={!!this.sidePanel} onClose={() => ctx.navigation.goto('.', {sidePanel: null})}>
                            {this.sidePanel === 'submit-new-workflow' && (
                                <WorkflowCreator
                                    namespace={Utils.getNamespaceWithDefault(this.state.namespace)}
                                    onCreate={wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`))}
                                />
                            )}
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchWorkflows(namespace: string, selectedPhases: WorkflowPhase[], selectedLabels: string[], pagination: Pagination): void {
        if (this.listWatch) {
            this.listWatch.stop();
        }
        this.listWatch = new ListWatch(
            () => services.workflows.list(namespace, selectedPhases, selectedLabels, pagination),
            (resourceVersion: string) => services.workflows.watchFields({namespace, phases: selectedPhases, labels: selectedLabels, resourceVersion}),
            metadata =>
                this.setState(
                    {
                        error: null,
                        namespace,
                        pagination: {offset: pagination.offset, limit: pagination.limit, nextOffset: metadata.continue},
                        selectedPhases,
                        selectedLabels,
                        selectedWorkflows: new Map<string, models.Workflow>()
                    },
                    this.saveHistory
                ),
            () => this.setState({error: null}),
            workflows => this.setState({workflows: workflows.slice(0, this.state.pagination.limit || 999999)}),
            error => this.setState({error}),
            sortByYouth
        );
        this.listWatch.start();
    }

    private changeFilters(namespace: string, selectedPhases: WorkflowPhase[], selectedLabels: string[], pagination: Pagination) {
        this.fetchWorkflows(namespace, selectedPhases, selectedLabels, pagination);
    }

    private saveHistory() {
        this.storage.setItem('options', this.options, {} as WorkflowListRenderOptions);
        const newNamespace = Utils.managedNamespace ? '' : this.state.namespace;
        this.url = uiUrl('workflows' + (newNamespace ? '/' + newNamespace : '') + '?' + this.filterParams.toString());
        Utils.currentNamespace = this.state.namespace;
    }

    private countsByCompleted() {
        const counts = {complete: 0, incomplete: 0};
        (this.state.workflows || []).forEach(wf => {
            if (wf.metadata.labels && wf.metadata.labels[labels.completed] === 'true') {
                counts.complete++;
            } else {
                counts.incomplete++;
            }
        });
        return counts;
    }

    private renderWorkflows() {
        const counts = this.countsByCompleted();
        return (
            <>
                <ErrorNotice error={this.state.error} />
                {!this.state.workflows ? (
                    <Loading />
                ) : this.state.workflows.length === 0 ? (
                    <ZeroState title='No workflows'>
                        <p>To create a new workflow, use the button above.</p>
                        <p>
                            <ExampleManifests />.
                        </p>
                    </ZeroState>
                ) : (
                    <>
                        {(counts.complete > 100 || counts.incomplete > 100) && (
                            <CostOptimisationNudge name='workflow-list'>
                                You have at least {counts.incomplete} incomplete, and {counts.complete} complete workflows. Reducing these amounts will reduce your costs.
                            </CostOptimisationNudge>
                        )}
                        <div className='argo-table-list'>
                            <div className='row argo-table-list__head'>
                                <div className='columns small-1 workflows-list__status'>
                                    <input
                                        type='checkbox'
                                        className='workflows-list__status--checkbox'
                                        checked={this.state.workflows.length === this.state.selectedWorkflows.size}
                                        onClick={e => {
                                            e.stopPropagation();
                                        }}
                                        onChange={e => {
                                            if (this.state.workflows.length === this.state.selectedWorkflows.size) {
                                                // All workflows are selected, deselect them all
                                                this.updateCurrentlySelectedAndBatchActions(new Map<string, models.Workflow>());
                                            } else {
                                                // Not all workflows are selected, select them all
                                                const currentlySelected: Map<string, Workflow> = this.state.selectedWorkflows;
                                                this.state.workflows.forEach(wf => {
                                                    if (!currentlySelected.has(wf.metadata.uid)) {
                                                        currentlySelected.set(wf.metadata.uid, wf);
                                                    }
                                                });
                                                this.updateCurrentlySelectedAndBatchActions(currentlySelected);
                                            }
                                        }}
                                    />
                                </div>
                                <div className='row small-11'>
                                    <div className='columns small-3'>NAME</div>
                                    <div className='columns small-1'>NAMESPACE</div>
                                    <div className='columns small-1'>STARTED</div>
                                    <div className='columns small-1'>FINISHED</div>
                                    <div className='columns small-1'>DURATION</div>
                                    <div className='columns small-1'>PROGRESS</div>
                                    <div className='columns small-2'>MESSAGE</div>
                                    <div className='columns small-1'>DETAILS</div>
                                </div>
                            </div>
                            {this.state.workflows.map(wf => {
                                return (
                                    <WorkflowsRow
                                        workflow={wf}
                                        key={wf.metadata.uid}
                                        checked={this.state.selectedWorkflows.has(wf.metadata.uid)}
                                        onChange={key => {
                                            const value = `${key}=${wf.metadata.labels[key]}`;
                                            let newTags: string[] = [];
                                            if (this.state.selectedLabels.indexOf(value) === -1) {
                                                newTags = this.state.selectedLabels.concat(value);
                                                this.setState({selectedLabels: newTags});
                                            }
                                            this.changeFilters(this.state.namespace, this.state.selectedPhases, newTags, this.state.pagination);
                                        }}
                                        select={subWf => {
                                            const wfUID = subWf.metadata.uid;
                                            if (!wfUID) {
                                                return;
                                            }
                                            const currentlySelected: Map<string, Workflow> = this.state.selectedWorkflows;
                                            if (!currentlySelected.has(wfUID)) {
                                                currentlySelected.set(wfUID, subWf);
                                            } else {
                                                currentlySelected.delete(wfUID);
                                            }
                                            this.updateCurrentlySelectedAndBatchActions(currentlySelected);
                                        }}
                                    />
                                );
                            })}
                        </div>
                        <PaginationPanel
                            onChange={pagination => this.changeFilters(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, pagination)}
                            pagination={this.state.pagination}
                            numRecords={(this.state.workflows || []).length}
                        />
                    </>
                )}
            </>
        );
    }

    private updateCurrentlySelectedAndBatchActions(newSelectedWorkflows: Map<string, Workflow>): void {
        const actions: any = Actions.WorkflowOperationsMap;
        const nowDisabled: any = {...allBatchActionsEnabled};
        for (const action of Object.keys(nowDisabled)) {
            for (const wf of Array.from(newSelectedWorkflows.values())) {
                nowDisabled[action] = nowDisabled[action] || actions[action].disabled(wf);
            }
        }
        this.setState({batchActionDisabled: nowDisabled, selectedWorkflows: new Map<string, models.Workflow>(newSelectedWorkflows)});
    }

    private renderQuery(ctx: any) {
        return (
            <Query>
                {q => (
                    <div>
                        <i className='fa fa-search' />
                        {q.get('search') && (
                            <i
                                className='fa fa-times'
                                onClick={() => {
                                    ctx.navigation.goto('.', {search: null}, {replace: true});
                                }}
                            />
                        )}
                        <Autocomplete
                            filterSuggestions={true}
                            renderInput={inputProps => (
                                <input
                                    {...inputProps}
                                    onFocus={e => {
                                        e.target.select();
                                        if (inputProps.onFocus) {
                                            inputProps.onFocus(e);
                                        }
                                    }}
                                    className='argo-field'
                                />
                            )}
                            renderItem={item => (
                                <React.Fragment>
                                    <i className='icon argo-icon-workflow' /> {item.label}
                                </React.Fragment>
                            )}
                            onSelect={val => {
                                ctx.navigation.goto(uiUrl(`workflows/${val}`));
                            }}
                            onChange={e => {
                                ctx.navigation.goto('.', {search: e.target.value}, {replace: true});
                            }}
                            value={q.get('search') || ''}
                            items={(this.state.workflows || []).map(wf => wf.metadata.namespace + '/' + wf.metadata.name)}
                        />
                    </div>
                )}
            </Query>
        );
    }
}

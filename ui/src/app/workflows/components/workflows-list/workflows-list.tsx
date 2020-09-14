import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {Subscription} from 'rxjs';

import {Autocomplete, Page, SlidingPanel} from 'argo-ui';
import * as models from '../../../../models';
import {labels, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';

import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Query} from '../../../shared/components/query';
import {ZeroState} from '../../../shared/components/zero-state';
import {exampleWorkflow} from '../../../shared/examples';
import {Utils} from '../../../shared/utils';
import * as Actions from '../../../shared/workflow-operations-map';

import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {PaginationPanel} from '../../../shared/components/pagination-panel';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {WorkflowFilters} from '../workflow-filters/workflow-filters';
import {WorkflowsRow} from '../workflows-row/workflows-row';
import {WorkflowsToolbar} from '../workflows-toolbar/workflows-toolbar';

require('./workflows-list.scss');

interface State {
    namespace: string;
    pagination: Pagination;
    selectedPhases: string[];
    selectedLabels: string[];
    selectedWorkflows: Map<string, models.Workflow>;
    workflows?: Workflow[];
    error?: Error;
    batchActionDisabled: Actions.OperationDisabled;
}

interface WorkflowListRenderOptions {
    paginationLimit: number;
    selectedPhases: string[];
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

const LOCAL_STORAGE_KEY = 'ListOptions';

export class WorkflowsList extends BasePage<RouteComponentProps<any>, State> {
    private get wfInput() {
        return Utils.tryJsonParse(this.queryParam('new'));
    }

    private static saveOptions(newChanges: WorkflowListRenderOptions) {
        localStorage.setItem(LOCAL_STORAGE_KEY, JSON.stringify(newChanges));
    }

    private static getOptions(): WorkflowListRenderOptions {
        if (localStorage.getItem(LOCAL_STORAGE_KEY) !== null) {
            return JSON.parse(localStorage.getItem(LOCAL_STORAGE_KEY)) as WorkflowListRenderOptions;
        }
        return {
            paginationLimit: 0,
            selectedPhases: [],
            selectedLabels: []
        } as WorkflowListRenderOptions;
    }

    private subscription: Subscription;

    constructor(props: RouteComponentProps<State>, context: any) {
        super(props, context);
        const savedOptions = WorkflowsList.getOptions();
        this.state = {
            pagination: {
                offset: this.queryParam('offset'),
                limit: parseLimit(this.queryParam('limit')) || savedOptions.paginationLimit
            },
            namespace: this.props.match.params.namespace || '',
            selectedPhases: this.queryParams('phase').length > 0 ? this.queryParams('phase') : savedOptions.selectedPhases,
            selectedLabels: this.queryParams('label').length > 0 ? this.queryParams('label') : savedOptions.selectedLabels,
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
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflows'
                        toolbar={{
                            breadcrumbs: [{title: 'Workflows', path: uiUrl('workflows')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Submit New Workflow',
                                        iconClassName: 'fa fa-plus',
                                        action: () => ctx.navigation.goto('.', {new: '{}'})
                                    }
                                ]
                            },
                            tools: []
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
                                <div>{this.renderQuery(ctx)}</div>
                                <div>
                                    <WorkflowFilters
                                        workflows={this.state.workflows || []}
                                        namespace={this.state.namespace}
                                        phaseItems={Object.values(models.NODE_PHASE)}
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
                        <SlidingPanel isShown={!!this.wfInput} onClose={() => ctx.navigation.goto('.', {new: null})}>
                            <ResourceEditor
                                title='Submit Workflow'
                                kind='Workflow'
                                upload={true}
                                editing={true}
                                value={exampleWorkflow(this.state.namespace)}
                                onSubmit={wfValue =>
                                    services.workflows
                                        .create(wfValue, wfValue.metadata.namespace || this.state.namespace)
                                        .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
                                }
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchWorkflows(namespace: string, selectedPhases: string[], selectedLabels: string[], pagination: Pagination): void {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
        services.workflows
            .list(namespace, selectedPhases, selectedLabels, pagination)
            .then(wfList => {
                this.setState(
                    {
                        error: null,
                        namespace,
                        workflows: wfList.items || [],
                        pagination: {offset: pagination.offset, limit: pagination.limit, nextOffset: wfList.metadata.continue},
                        selectedPhases,
                        selectedLabels,
                        selectedWorkflows: new Map<string, models.Workflow>()
                    },
                    this.saveHistory
                );
                return wfList.metadata.resourceVersion;
            })
            .then(resourceVersion => {
                this.subscription = services.workflows
                    .watchFields({namespace, phases: selectedPhases, labels: selectedLabels, resourceVersion})
                    .map(workflowChange => {
                        const workflows = this.state.workflows;
                        if (!workflowChange) {
                            return {workflows, updated: false};
                        }
                        const index = workflows.findIndex(item => item.metadata.uid === workflowChange.object.metadata.uid);
                        if (index > -1 && workflowChange.object.metadata.resourceVersion === workflows[index].metadata.resourceVersion) {
                            return {workflows, updated: false};
                        }
                        if (workflowChange.type === 'DELETED') {
                            if (index > -1) {
                                workflows.splice(index, 1);
                            }
                        } else {
                            if (index > -1) {
                                workflows[index] = workflowChange.object;
                            } else if (!this.state.pagination.limit) {
                                workflows.unshift(workflowChange.object);
                            }
                        }
                        return {workflows, updated: true};
                    })
                    .filter(item => item.updated)
                    .map(item => item.workflows)
                    .subscribe(
                        workflows => this.setState({error: null, workflows}),
                        error => this.setState({error})
                    );
            })
            .then(_ => this.setState({error: null}))
            .catch(error => this.setState({error}));
    }

    private changeFilters(namespace: string, selectedPhases: string[], selectedLabels: string[], pagination: Pagination) {
        this.fetchWorkflows(namespace, selectedPhases, selectedLabels, pagination);
    }

    private get filterParams() {
        const params = new URLSearchParams();
        this.state.selectedPhases.forEach(phase => {
            params.append('phase', phase);
        });
        this.state.selectedLabels.forEach(label => {
            params.append('label', label);
        });
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
        options.selectedLabels = [];
        this.state.selectedLabels.forEach(label => {
            options.selectedLabels.push(label);
        });
        if (this.state.pagination.limit) {
            options.paginationLimit = this.state.pagination.limit;
        }
        return options;
    }

    private saveHistory() {
        WorkflowsList.saveOptions(this.options);
        this.url = uiUrl('workflows/' + this.state.namespace || '' + '?' + this.filterParams.toString());
        Utils.setCurrentNamespace(this.state.namespace);
    }

    private countsByCompleted() {
        const counts = {complete: 0, incomplete: 0};
        this.state.workflows.forEach(wf => {
            if (wf.metadata.labels && wf.metadata.labels[labels.completed] === 'true') {
                counts.complete++;
            } else {
                counts.incomplete++;
            }
        });
        return counts;
    }

    private renderWorkflows() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.workflows) {
            return <Loading />;
        }
        if (this.state.workflows.length === 0) {
            return (
                <ZeroState title='No workflows'>
                    <p>To create a new workflow, use the button above.</p>
                </ZeroState>
            );
        }

        const counts = this.countsByCompleted();

        return (
            <>
                {(counts.complete > 100 || counts.incomplete > 100) && (
                    <CostOptimisationNudge name='workflow-list'>
                        You have at least {counts.incomplete} incomplete, and {counts.complete} complete workflows. Reducing these amounts will reduce your costs.
                    </CostOptimisationNudge>
                )}
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns workflows-list__status small-1' />
                        <div className='row small-11'>
                            <div className='columns small-3'>NAME</div>
                            <div className='columns small-2'>NAMESPACE</div>
                            <div className='columns small-2'>STARTED</div>
                            <div className='columns small-2'>FINISHED</div>
                            <div className='columns small-1'>DURATION</div>
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
                />
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

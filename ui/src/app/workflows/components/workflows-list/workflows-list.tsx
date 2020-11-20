import {Autocomplete, Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {labels, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';

import {BasePage} from '../../../shared/components/base-page';

import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {Loading} from '../../../shared/components/loading';
import {PaginationPanel} from '../../../shared/components/pagination-panel';
import {Query} from '../../../shared/components/query';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {exampleWorkflow} from '../../../shared/examples';
import {ListWatch, sortByYouth} from '../../../shared/list-watch';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import * as Actions from '../../../shared/workflow-operations-map';
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
    resourceVersion?: string;
    error?: Error;
    batchActionDisabled: Actions.OperationDisabled;
}

interface WorkflowListRenderOptions {
    paginationLimit: number;
    selectedPhases: string[];
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
        if (this.state.pagination.limit) {
            options.paginationLimit = this.state.pagination.limit;
        }
        return options;
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
            selectedPhases: []
        } as WorkflowListRenderOptions;
    }

    private listWatch: ListWatch<Workflow>;

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
            selectedLabels: this.queryParams('label'),
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
                                namespace={this.state.namespace || 'default'}
                                value={exampleWorkflow()}
                                onSubmit={wfValue =>
                                    services.workflows
                                        .create(wfValue, wfValue.metadata.namespace || this.state.namespace)
                                        .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
                                }
                            />
                            <p>
                                <ExampleManifests />.
                            </p>
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchWorkflows(namespace: string, selectedPhases: string[], selectedLabels: string[], pagination: Pagination): void {
        if (this.listWatch) {
            this.listWatch.stop();
        }
        this.listWatch = new ListWatch(
            () => services.workflows.list(namespace, selectedPhases, selectedLabels, pagination),
            (resourceVersion: string) => services.workflows.watchFields({namespace, phases: selectedPhases, labels: selectedLabels, resourceVersion}),
            metadata =>
                this.setState(
                    {
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

    private changeFilters(namespace: string, selectedPhases: string[], selectedLabels: string[], pagination: Pagination) {
        this.fetchWorkflows(namespace, selectedPhases, selectedLabels, pagination);
    }

    private saveHistory() {
        WorkflowsList.saveOptions(this.options);
        this.url = uiUrl('workflows/' + this.state.namespace || '' + '?' + this.filterParams.toString());
        Utils.setCurrentNamespace(this.state.namespace);
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
                {this.state.error && <ErrorNotice error={this.state.error} />}
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
                                <div className='columns workflows-list__status small-1' />
                                <div className='row small-11'>
                                    <div className='columns small-3'>NAME</div>
                                    <div className='columns small-2'>NAMESPACE</div>
                                    <div className='columns small-2'>STARTED</div>
                                    <div className='columns small-2'>FINISHED</div>
                                    <div className='columns small-1'>DURATION</div>
                                    <div className='columns small-1'>PROGRESS</div>
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

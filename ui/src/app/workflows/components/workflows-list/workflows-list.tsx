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
import {ResourceSubmit} from '../../../shared/components/resource-submit';
import {ZeroState} from '../../../shared/components/zero-state';
import {exampleWorkflow} from '../../../shared/examples';
import {Utils} from '../../../shared/utils';
import * as Actions from '../../../shared/workflow-operations';

import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {PaginationPanel} from '../../../shared/components/pagination-panel';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {WorkflowFilters} from '../workflow-filters/workflow-filters';
import {WorkflowsRow} from '../workflows-row/workflows-row';
import {WorkflowsToolbar} from '../workflows-toolbar/workflows-toolbar';

require('./workflows-list.scss');

interface State {
    pagination: Pagination;
    loading: boolean;
    initialized: boolean;
    managedNamespace: boolean;
    namespace: string;
    selectedPhases: string[];
    selectedLabels: string[];
    selectedWorkflows: {[index: string]: models.Workflow};
    workflows?: Workflow[];
    error?: Error;
    batchActionDisabled: Actions.OperationDisabled;
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
    private get wfInput() {
        return Utils.tryJsonParse(this.queryParam('new'));
    }

    private subscription: Subscription;

    constructor(props: RouteComponentProps<State>, context: any) {
        super(props, context);
        this.state = {
            loading: true,
            pagination: {offset: this.queryParam('offset'), limit: parseLimit(this.queryParam('limit'))},
            initialized: false,
            managedNamespace: false,
            namespace: this.props.match.params.namespace || Utils.getCurrentNamespace() || '',
            selectedPhases: this.queryParams('phase'),
            selectedLabels: this.queryParams('label'),
            selectedWorkflows: {},
            batchActionDisabled: {...allBatchActionsEnabled}
        };
    }

    public componentDidMount(): void {
        this.fetchWorkflows(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, this.state.pagination);
        this.setState({selectedWorkflows: {}});
    }

    public componentWillUnmount(): void {
        this.setState({selectedWorkflows: {}});
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
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
                            loadWorkflows={() => {
                                this.setState({selectedWorkflows: {}});
                                this.fetchWorkflows(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, {limit: this.state.pagination.limit});
                            }}
                            isDisabled={this.state.batchActionDisabled}
                        />
                        <div className='row'>
                            <div className='columns small-12 xlarge-2'>
                                <div>{this.renderQuery(ctx)}</div>
                                <div>
                                    <WorkflowFilters
                                        workflows={this.state.workflows}
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
                            <ResourceSubmit<models.Workflow>
                                resourceName={'Workflow'}
                                defaultResource={exampleWorkflow(this.state.namespace)}
                                validate={wfValue => {
                                    if (!wfValue || !wfValue.metadata) {
                                        return {valid: false, message: 'Invalid Workflow: metadata cannot be blank'};
                                    }
                                    wfValue.metadata.namespace = wfValue.metadata.namespace || this.state.namespace;
                                    if (!wfValue.metadata.namespace) {
                                        return {valid: false, message: 'Invalid Workflow: metadata.namespace cannot be blank'};
                                    }
                                    return {valid: true};
                                }}
                                onSubmit={wfValue => {
                                    return services.workflows
                                        .create(wfValue, wfValue.metadata.namespace || this.state.namespace)
                                        .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)));
                                }}
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
        let workflowList;
        let newNamespace = namespace;
        if (!this.state.initialized) {
            workflowList = services.info.getInfo().then(info => {
                if (info.managedNamespace) {
                    newNamespace = info.managedNamespace;
                }
                this.setState({initialized: true, managedNamespace: !!info.managedNamespace});
                return services.workflows.list(newNamespace, selectedPhases, selectedLabels, pagination);
            });
        } else {
            if (this.state.managedNamespace) {
                newNamespace = this.state.namespace;
            }
            workflowList = services.workflows.list(newNamespace, selectedPhases, selectedLabels, pagination);
        }
        workflowList
            .then(wfList => {
                this.setState({
                    workflows: wfList.items || [],
                    pagination: {offset: pagination.offset, limit: pagination.limit, nextOffset: wfList.metadata.continue},
                    namespace: newNamespace,
                    selectedPhases,
                    selectedLabels,
                    selectedWorkflows: {}
                });
                Utils.setCurrentNamespace(newNamespace);
                return wfList.metadata.resourceVersion;
            })
            .then(resourceVersion => {
                this.subscription = services.workflows
                    .watchFields({namespace: newNamespace, phases: selectedPhases, labels: selectedLabels, resourceVersion})
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
                    .catch((error, caught) => caught)
                    .subscribe(workflows => this.setState({workflows}));
            })
            .then(_ => this.setState({loading: false}))
            .catch(error => this.setState({error, loading: false}));
    }

    private changeFilters(namespace: string, selectedPhases: string[], selectedLabels: string[], pagination: Pagination) {
        const params = new URLSearchParams();
        selectedPhases.forEach(phase => {
            params.append('phase', phase);
        });
        selectedLabels.forEach(label => {
            params.append('label', label);
        });
        if (pagination.offset) {
            params.append('offset', pagination.offset);
        }
        if (pagination.limit) {
            params.append('limit', pagination.limit.toString());
        }
        const url = 'workflows/' + namespace + '?' + params.toString();
        history.pushState(null, '', uiUrl(url));
        this.fetchWorkflows(namespace, selectedPhases, selectedLabels, pagination);
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
                                    const currentlySelected = this.state.selectedWorkflows;
                                    if (!(wfUID in currentlySelected)) {
                                        this.updateBatchActionsDisabled(subWf, false);
                                        currentlySelected[wfUID] = subWf;
                                    } else {
                                        this.updateBatchActionsDisabled(subWf, true);
                                        delete currentlySelected[wfUID];
                                    }
                                    this.setState({selectedWorkflows: {...currentlySelected}});
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

    private updateBatchActionsDisabled(wf: Workflow, deselect: boolean): void {
        const currentlyDisabled: any = this.state.batchActionDisabled;
        const actions: any = Actions.WorkflowOperations;
        const nowDisabled: any = {...allBatchActionsEnabled};
        for (const action of Object.keys(currentlyDisabled)) {
            if (deselect) {
                for (const wfUID of Object.keys(this.state.selectedWorkflows)) {
                    if (wfUID === wf.metadata.uid) {
                        continue;
                    }
                    nowDisabled[action] = actions[action].disabled(this.state.selectedWorkflows[wfUID]) || nowDisabled[action];
                }
            } else {
                nowDisabled[action] = actions[action].disabled(wf) || currentlyDisabled[action];
            }
        }
        this.setState({batchActionDisabled: nowDisabled});
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
                                ctx.navigation.goto(`./${val}`);
                            }}
                            onChange={e => {
                                ctx.navigation.goto('.', {search: e.target.value}, {replace: true});
                            }}
                            value={q.get('search') || ''}
                            items={this.state.workflows.map(wf => wf.metadata.namespace + '/' + wf.metadata.name)}
                        />
                    </div>
                )}
            </Query>
        );
    }
}

import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Subscription} from 'rxjs';

import {Autocomplete, Page, SlidingPanel} from 'argo-ui';
import * as models from '../../../../models';
import {Workflow} from '../../../../models';
import {compareWorkflows} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';

import {WorkflowListItem} from '..';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Query} from '../../../shared/components/query';
import {ResourceSubmit} from '../../../shared/components/resource-submit';
import {ZeroState} from '../../../shared/components/zero-state';
import {exampleWorkflow} from '../../../shared/examples';
import {Utils} from '../../../shared/utils';

import {WorkflowFilters} from '../workflow-filters/workflow-filters';

require('./workflows-list.scss');

interface State {
    loading: boolean;
    initialized: boolean;
    managedNamespace: boolean;
    namespace: string;
    selectedPhases: string[];
    selectedLabels: string[];
    workflows?: Workflow[];
    error?: Error;
}

export class WorkflowsList extends BasePage<RouteComponentProps<any>, State> {
    private get wfInput() {
        return Utils.tryJsonParse(this.queryParam('new'));
    }
    private subscription: Subscription;

    constructor(props: RouteComponentProps<State>, context: any) {
        super(props, context);
        this.state = {
            loading: true,
            initialized: false,
            managedNamespace: false,
            namespace: this.props.match.params.namespace || Utils.getCurrentNamespace() || '',
            selectedPhases: this.queryParams('phase'),
            selectedLabels: this.queryParams('label')
        };
    }

    public componentDidMount(): void {
        this.fetchWorkflows(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels);
    }

    public componentWillUnmount(): void {
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
                                        onChange={(namespace, selectedPhases, selectedLabels) => this.changeFilters(namespace, selectedPhases, selectedLabels)}
                                    />
                                </div>
                            </div>
                            <div className='columns small-12 xlarge-10'>{this.renderWorkflows(ctx)}</div>
                        </div>
                        <SlidingPanel isShown={!!this.wfInput} onClose={() => ctx.navigation.goto('.', {new: null})}>
                            <ResourceSubmit<models.Workflow>
                                resourceName={'Workflow'}
                                defaultResource={exampleWorkflow(this.state.namespace)}
                                validate={wfValue => {
                                    if (!wfValue || !wfValue.metadata) {
                                        return {valid: false, message: 'Invalid Workflow definition'};
                                    }
                                    if (wfValue.metadata.namespace === undefined || wfValue.metadata.namespace === '') {
                                        return {valid: false, message: 'Namespace is missing'};
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

    private fetchWorkflows(namespace: string, selectedPhases: string[], selectedLabels: string[]): void {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
        let workflowList;
        let newNamespace = namespace;
        if (!this.state.initialized) {
            workflowList = services.info.get().then(info => {
                if (info.managedNamespace) {
                    newNamespace = info.managedNamespace;
                }
                this.setState({initialized: true, managedNamespace: info.managedNamespace ? true : false});
                return services.workflows.list(newNamespace, selectedPhases, selectedLabels);
            });
        } else {
            if (this.state.managedNamespace) {
                newNamespace = this.state.namespace;
            }
            workflowList = services.workflows.list(newNamespace, selectedPhases, selectedLabels);
        }
        workflowList
            .then(list => list.items)
            .then(list => list || [])
            .then(workflows => {
                this.setState({workflows, namespace: newNamespace, selectedPhases, selectedLabels});
                Utils.setCurrentNamespace(newNamespace);
            })
            .then(() => {
                this.subscription = services.workflows
                    .watch({namespace: newNamespace, phases: selectedPhases, labels: selectedLabels})
                    .map(workflowChange => {
                        const workflows = this.state.workflows;
                        if (!workflowChange) {
                            return {workflows, updated: false};
                        }
                        const index = workflows.findIndex(item => item.metadata.name === workflowChange.object.metadata.name);
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
                            } else {
                                workflows.unshift(workflowChange.object);
                            }
                        }
                        return {workflows, updated: true};
                    })
                    .filter(item => item.updated)
                    .map(item => item.workflows)
                    .catch((error, caught) => {
                        return caught;
                    })
                    .subscribe(workflows => this.setState({workflows}));
            })
            .then(_ => this.setState({loading: false}))
            .catch(error => this.setState({error, loading: false}));
    }

    private changeFilters(namespace: string, selectedPhases: string[], selectedLabels: string[]) {
        const params = new URLSearchParams();
        selectedPhases.forEach(phase => {
            params.append('phase', phase);
        });
        selectedLabels.forEach(label => {
            params.append('label', label);
        });
        let url = 'workflows/' + namespace;
        if (selectedPhases.length > 0 || selectedLabels.length > 0) {
            url += '?' + params.toString();
        }
        history.pushState(null, '', uiUrl(url));
        this.fetchWorkflows(namespace, selectedPhases, selectedLabels);
    }

    private renderWorkflows(ctx: any) {
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
        this.state.workflows.sort(compareWorkflows);

        return (
            <>
                <div className='row'>
                    <div className='columns small-12 xxlarge-12'>
                        {this.state.workflows.map(workflow => (
                            <div key={workflow.metadata.name}>
                                <Link to={uiUrl(`workflows/${workflow.metadata.namespace}/${workflow.metadata.name}`)}>
                                    <WorkflowListItem workflow={workflow} archived={false} />
                                </Link>
                            </div>
                        ))}
                    </div>
                </div>
            </>
        );
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

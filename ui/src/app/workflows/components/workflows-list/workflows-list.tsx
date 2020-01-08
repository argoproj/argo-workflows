import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Observable} from 'rxjs';

import {Autocomplete, Page, SlidingPanel, TopBarFilter} from 'argo-ui';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';

import {WorkflowListItem} from '..';
import {Workflow} from '../../../../models';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {Query} from '../../../shared/components/query';
import {YamlEditor} from '../../../shared/components/yaml-editor/yaml-editor';
import {ZeroState} from '../../../shared/components/zero-state';
import {Utils} from '../../../shared/utils';

require('./workflows-list.scss');

const placeholderWorkflow = (namespace:string) =>  `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: ${namespace}
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`;

interface State {
    workflows?: Workflow[];
    error?: Error;
}

export class WorkflowsList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.queryParam('namespace') || '';
    }

    private set namespace(namespace: string) {
        this.setQueryParams({namespace});
    }

    private get phases() {
        return this.queryParams('phase');
    }

    private get wfInput() {
        return Utils.tryJsonParse(this.queryParam('new'));
    }

    constructor(props: RouteComponentProps<State>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        this.loadWorkflows(this.namespace, this.phases);
    }

    public render() {
        if (this.state.error) {
            throw this.state.error;
        }
        const filter: TopBarFilter<string> = {
            items: Object.keys(models.NODE_PHASE).map(phase => ({
                value: (models.NODE_PHASE as any)[phase],
                label: (models.NODE_PHASE as any)[phase]
            })),
            selectedValues: this.phases,
            selectionChanged: phases => {
                const query = phases.length > 0 ? '?' + phases.map(phase => `phase=${phase}`).join('&') : '';
                this.appContext.router.history.push(uiUrl(`workflows${query}`));
                this.loadWorkflows(this.namespace, phases);
            }
        };
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflows'
                        toolbar={{
                            filter,
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
                            tools: [
                                <NamespaceFilter
                                    key='namespace-filter'
                                    value={this.namespace}
                                    onChange={namespace => {
                                        this.namespace = namespace;
                                        this.loadWorkflows(namespace, this.phases);
                                    }}
                                />
                            ]
                        }}>
                        <div>{this.renderWorkflows(ctx)}</div>
                        <SlidingPanel isShown={!!this.wfInput} onClose={() => ctx.navigation.goto('.', {new: null})}>
                            Submit Workflow
                            <YamlEditor
                                minHeight={800}
                                initialEditMode={true}
                                submitMode={true}
                                placeHolder={placeholderWorkflow(this.namespace || "default")}
                                onSave={rawWf => {
                                    return services.workflows
                                        .create(JSON.parse(rawWf))
                                        .then()
                                        .then(wf => ctx.navigation.goto(`/workflows/${wf.metadata.namespace}/${wf.metadata.name}`));
                                }}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private loadWorkflows(namespace: string, phases: string[]) {
        Observable.fromPromise(services.workflows.list(phases, namespace).catch(error => this.setState({error})))
            .flatMap((workflows: Workflow[]) =>
                Observable.merge(
                    Observable.from([workflows]),
                    services.workflows
                        .watch({namespace, phases})
                        .map(workflowChange => {
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
                            this.setState({error});
                            return caught;
                        })
                )
            )
            .subscribe(workflows => {
                this.setState({workflows});
            });
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

        return (
            <>
                <div className='row'>
                    <div className='columns small-12 xxlarge-12'>{this.renderQuery(ctx)}</div>
                </div>
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

import {Page, SlidingPanel} from 'argo-ui';

import * as classNames from 'classnames';
import {isNaN} from 'formik';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {ResourceSubmit} from '../../../shared/components/resource-submit';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {exampleWorkflow} from '../../../shared/examples';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {WorkflowFilters} from '../../../workflows/components/workflow-filters/workflow-filters';

interface State {
    offset: number;
    nextOffset: number;
    loading: boolean;
    initialized: boolean;
    managedNamespace: boolean;
    namespace: string;
    selectedPhases: string[];
    selectedLabels: string[];
    workflows?: Workflow[];
    error?: Error;
}

export class ArchivedWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private get wfInput() {
        return Utils.tryJsonParse(this.queryParam('new'));
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {
            loading: true,
            offset: this.parseOffset(this.queryParam('continue') || ''),
            nextOffset: 0,
            initialized: false,
            managedNamespace: false,
            namespace: this.props.match.params.namespace || Utils.getCurrentNamespace() || '',
            selectedPhases: this.queryParams('phase'),
            selectedLabels: this.queryParams('label')
        };
    }

    public componentDidMount(): void {
        this.fetchArchivedWorkflows(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, this.state.offset);
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
                            breadcrumbs: [{title: 'Archived Workflows', path: uiUrl('archived-workflows')}],
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
                                <div>
                                    <WorkflowFilters
                                        workflows={this.state.workflows}
                                        namespace={this.state.namespace}
                                        phaseItems={Object.values([models.NODE_PHASE.SUCCEEDED, models.NODE_PHASE.FAILED, models.NODE_PHASE.ERROR])}
                                        selectedPhases={this.state.selectedPhases}
                                        selectedLabels={this.state.selectedLabels}
                                        onChange={(namespace, selectedPhases, selectedLabels) => this.changeFilters(namespace, selectedPhases, selectedLabels, 0)}
                                    />
                                </div>
                            </div>
                            <div className='columns small-12 xlarge-10'>{this.renderWorkflows()}</div>
                        </div>
                        <SlidingPanel isShown={!!this.wfInput} onClose={() => ctx.navigation.goto('.', {new: null})}>
                            <ResourceSubmit<models.Workflow>
                                resourceName={'Workflow'}
                                defaultResource={exampleWorkflow(this.state.namespace)}
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

    private parseOffset(str: string) {
        if (isNaN(str)) {
            return 0;
        }
        const result = parseInt(str, 10);
        return result >= 0 ? result : 0;
    }

    private changeFilters(namespace: string, selectedPhases: string[], selectedLabels: string[], offset: number) {
        const params = new URLSearchParams();
        selectedPhases.forEach(phase => {
            params.append('phase', phase);
        });
        selectedLabels.forEach(label => {
            params.append('label', label);
        });
        if (offset > 0) {
            params.append('continue', offset.toString());
        }
        let url = 'archived-workflows/' + namespace;
        if (selectedPhases.length > 0 || selectedLabels.length > 0 || offset > 0) {
            url += '?' + params.toString();
        }
        history.pushState(null, '', uiUrl(url));
        this.fetchArchivedWorkflows(namespace, selectedPhases, selectedLabels, offset && offset >= 0 ? offset : 0);
    }

    private fetchArchivedWorkflows(namespace: string, selectedPhases: string[], selectedLabels: string[], offset: number): void {
        let archivedWorkflowList;
        let newNamespace = namespace;
        if (!this.state.initialized) {
            archivedWorkflowList = services.info.get().then(info => {
                if (info.managedNamespace) {
                    newNamespace = info.managedNamespace;
                }
                this.setState({initialized: true, managedNamespace: info.managedNamespace ? true : false});
                return services.archivedWorkflows.list(newNamespace, selectedPhases, selectedLabels, offset);
            });
        } else {
            if (this.state.managedNamespace) {
                newNamespace = this.state.namespace;
            }
            archivedWorkflowList = services.archivedWorkflows.list(newNamespace, selectedPhases, selectedLabels, offset);
        }
        archivedWorkflowList
            .then(list => {
                this.setState({
                    namespace: newNamespace,
                    workflows: list.items || [],
                    selectedPhases,
                    selectedLabels,
                    offset,
                    nextOffset: this.parseOffset(list.metadata.continue || ''),
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
                        <div className='columns small-5'>NAME</div>
                        <div className='columns small-3'>NAMESPACE</div>
                        <div className='columns small-3'>CREATED</div>
                    </div>
                    {this.state.workflows.map(w => (
                        <Link className='row argo-table-list__row' key={`${w.metadata.uid}`} to={uiUrl(`archived-workflows/${w.metadata.namespace}/${w.metadata.uid}`)}>
                            <div className='columns small-1'>
                                <i className={classNames('fa', Utils.statusIconClasses(w.status.phase))} />
                            </div>
                            <div className='columns small-5'>{w.metadata.name}</div>
                            <div className='columns small-3'>{w.metadata.namespace}</div>
                            <div className='columns small-3'>
                                <Timestamp date={w.metadata.creationTimestamp} />
                            </div>
                        </Link>
                    ))}
                </div>
                <p>
                    {this.state.offset !== 0 && (
                        <button
                            className='argo-button argo-button--base-o'
                            onClick={() => {
                                this.changeFilters(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, 0);
                            }}>
                            <i className='fa fa-chevron-left' /> Start
                        </button>
                    )}
                    {this.state.nextOffset !== 0 && (
                        <button
                            className='argo-button argo-button--base-o'
                            onClick={() => {
                                this.changeFilters(this.state.namespace, this.state.selectedPhases, this.state.selectedLabels, this.state.nextOffset);
                            }}>
                            Next: {this.state.nextOffset} <i className='fa fa-chevron-right' />
                        </button>
                    )}
                </p>
                <p>
                    <i className='fa fa-info-circle' /> Records are created in the archive when a workflow completes. {learnMore}.
                </p>
            </>
        );
    }
}

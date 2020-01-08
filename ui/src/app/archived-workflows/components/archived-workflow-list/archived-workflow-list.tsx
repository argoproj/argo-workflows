import {Page} from 'argo-ui';

import * as classNames from 'classnames';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

interface State {
    workflows?: Workflow[];
    error?: Error;
}

export class ArchivedWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private get continue() {
        return this.queryParam('continue') || '';
    }

    private set continue(continueArg: string) {
        this.setQueryParams({continue: continueArg});
    }

    private get namespace() {
        return this.queryParam('namespace') || '';
    }

    private set namespace(namespace: string) {
        this.setQueryParams({namespace});
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        this.loadArchivedWorkflows(this.namespace, this.continue);
    }

    public render() {
        if (this.state.error) {
            throw this.state.error;
        }

        return (
            <Page
                title='Archived Workflows'
                toolbar={{
                    breadcrumbs: [{title: 'Archived Workflows', path: uiUrl('archived-workflow')}],
                    tools: [
                        <NamespaceFilter
                            key='namespace-filter'
                            value={this.namespace}
                            onChange={namespace => {
                                this.namespace = namespace;
                                this.loadArchivedWorkflows(namespace, '');
                            }}
                        />
                    ]
                }}>
                <div className='row'>
                    <div className='columns small-12 xxlarge-2'>{this.renderWorkflows()}</div>
                </div>
            </Page>
        );
    }

    private loadArchivedWorkflows(namespace: string, continueArg: string) {
        services.archivedWorkflows
            .list(namespace, continueArg)
            .then(list => {
                this.continue = list.metadata.continue || '';
                this.setState({workflows: list.items || []});
            })
            .catch(error => this.setState({error}));
    }

    private renderWorkflows() {
        if (!!!this.state.workflows) {
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/apiserverimpl/docs/workflow-archive.md'>Learn more</a>;
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
                        <Link
                            className='row argo-table-list__row'
                            key={`${w.metadata.namespace}/${w.metadata.uid}`}
                            to={uiUrl(`archived-workflows/${w.metadata.namespace}/${w.metadata.uid}`)}>
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
                    {this.continue !== '' && (
                        <button
                            className='argo-button argo-button--base-o'
                            onClick={() => {
                                this.loadArchivedWorkflows(this.namespace, '');
                            }}>
                            <i className='fa fa-chevron-left' /> Start
                        </button>
                    )}
                    {this.continue !== '' && (
                        <button className='argo-button argo-button--base-o' onClick={() => this.loadArchivedWorkflows(this.namespace, this.continue)}>
                            Next: {this.continue} <i className='fa fa-chevron-right' />
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

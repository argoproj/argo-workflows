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
    continue: string;
    loading: boolean;
    namespace: string;
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
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.setState({namespace});
        history.pushState(null, '', uiUrl('archived-workflows/' + namespace));
        this.fetchArchivedWorkflows();
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {continue: '', loading: true, namespace: this.props.match.params.namespace || ''};
    }

    public componentWillMount(): void {
        this.fetchArchivedWorkflows();
    }

    public render() {
        if (this.state.loading) {
            return <Loading />;
        }
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
                            }}
                        />
                    ]
                }}>
                <div className='row'>
                    <div className='columns small-12'>{this.renderWorkflows()}</div>
                </div>
            </Page>
        );
    }

    private fetchArchivedWorkflows(): void {
        services.info
            .get()
            .then(info => {
                if (info.managedNamespace && info.managedNamespace !== this.namespace) {
                    this.namespace = info.managedNamespace;
                }
                return services.archivedWorkflows.list(this.namespace, this.continue);
            })
            .then(list => {
                this.setState({workflows: list.items || [], continue: list.metadata.continue || '', loading: false});
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
                    {this.continue !== '' && (
                        <button className='argo-button argo-button--base-o' onClick={() => (this.continue = '')}>
                            <i className='fa fa-chevron-left' /> Start
                        </button>
                    )}
                    {this.state.continue !== '' && (
                        <button className='argo-button argo-button--base-o' onClick={() => (this.continue = this.state.continue)}>
                            Next: {this.state.continue} <i className='fa fa-chevron-right' />
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

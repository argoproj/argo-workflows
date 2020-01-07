import {Page} from 'argo-ui';

import * as classNames from 'classnames';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {searchToMetadataFilter} from '../../../shared/filter';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

interface State {
    namespace: string;
    continue: string;
    workflows?: Workflow[];
    error?: Error;
}

export class ArchivedWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private get search() {
        return this.queryParam('search') || '';
    }

    private set search(search) {
        this.setQueryParams({search});
    }
    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {namespace: '', continue: ''};
    }

    public componentDidMount(): void {
        this.loadArchivedWorkflows();
    }

    public render() {
        if (this.state.error) {
            throw this.state.error;
        }

        return (
            <Page
                title='Archived Workflows'
                toolbar={{
                    breadcrumbs: [{title: 'Archived Workflows', path: uiUrl('archived-workflow')}]
                }}>
                <div className='row'>
                    <div className='columns small-12 xxlarge-2'>{this.renderWorkflows()}</div>
                </div>
            </Page>
        );
    }

    private loadArchivedWorkflows() {
        services.archivedWorkflows
            .list(this.state.namespace, this.state.continue)
            .then(list => this.setState({continue: list.metadata.continue, workflows: list.items}))
            .catch(error => this.setState({error}));
    }

    private renderWorkflows() {
        if (!this.state.workflows) {
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/apiserverimpl/docs/workflow-archive.md'>Learn more</a>;
        if (this.state.workflows.length === 0) {
            return (
                <div className='white-box'>
                    <h4>No archived workflows</h4>
                    <p>To add entries to the archive you must enabled archiving in configuration. Records are the created in the archive on workflow completion.</p>
                    <p>{learnMore}.</p>
                </div>
            );
        }

        const filter = searchToMetadataFilter(this.search);
        const workflows = this.state.workflows.filter(w => filter(w.metadata));
        return (
            <>
                <p>
                    <i className='fa fa-search' />
                    <input
                        className='argo-field'
                        defaultValue={this.search}
                        onChange={e => {
                            this.search = e.target.value;
                        }}
                        placeholder='e.g. name:hello-world namespace:argo'
                    />
                </p>
                {workflows.length === 0 ? (
                    <p>No archived workflows found</p>
                ) : (
                    <>
                        <div className='argo-table-list'>
                            <div className='row argo-table-list__head'>
                                <div className='columns small-1' />
                                <div className='columns small-5'>NAME</div>
                                <div className='columns small-3'>NAMESPACE</div>
                                <div className='columns small-3'>CREATED</div>
                            </div>
                            {workflows.map(w => (
                                <Link className='row argo-table-list__row' key={w.metadata.name} to={uiUrl(`archived-workflows/${w.metadata.namespace}/${w.metadata.uid}`)}>
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
                            {this.state.continue !== '' && (
                                <button className='argo-button argo-button--base-o' onClick={() => this.loadArchivedWorkflows()}>
                                    Continue: {this.state.continue} <i className='fa fa-chevron-right' />
                                </button>
                            )}
                        </p>
                        <p>
                            <i className='fa fa-info-circle' /> Records are created in the archive when a workflow completes. {learnMore}.
                        </p>
                    </>
                )}
            </>
        );
    }
}

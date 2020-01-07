import {MockupList, Page} from 'argo-ui';

import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {services} from '../../../shared/services';
import {WorkflowListItem} from '../../../workflows/components';

interface State {
    workflows?: Workflow[];
}

export class ArchivedWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        services.archivedWorkflows.list().then(workflows => {
            this.setState({workflows});
        });
    }

    private get search() {
        return this.queryParam('search') || '';
    }

    private set search(search) {
        this.setQueryParams({search});
    }

    public render() {
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

    private renderWorkflows() {
        if (this.state.workflows === undefined) {
            return <MockupList height={150} marginTop={30} />;
        }
        if (this.state.workflows.length === 0) {
            return (
                <div className='white-box'>
                    <h4>No archived workflows</h4>
                    <p>To record entries you must enabled archiving in configuration.</p>
                </div>
            );
        }

        const workflows = this.state.workflows.filter(workflow => workflow.metadata.name.indexOf(this.search) >= 0);
        return (
            <>
                <p>
                    <i className='fa fa-search' />
                    <input
                        className={'argo-field'}
                        defaultValue={this.search}
                        onChange={e => {
                            this.search = e.target.value;
                        }}
                    />
                </p>
                {workflows.length === 0 ? (
                    <p>No archived workflows found</p>
                ) : (
                    <>
                        {workflows.map(workflow => (
                            <div key={workflow.metadata.uid}>
                                <Link to={uiUrl(`archived-workflows/${workflow.metadata.namespace}/${workflow.metadata.uid}`)}>
                                    <WorkflowListItem workflow={workflow} archived={true} />
                                </Link>
                            </div>
                        ))}
                        <p>
                            <i className='fa fa-info-circle' /> Records are created in the archive when a workflow completes.
                        </p>
                    </>
                )}
            </>
        );
    }
}

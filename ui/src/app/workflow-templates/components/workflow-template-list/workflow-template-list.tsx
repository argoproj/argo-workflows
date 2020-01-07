import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Timestamp} from '../../../shared/components/timestamp';
import {services} from '../../../shared/services';

require('./workflow-template-list.scss');

interface State {
    templates?: models.WorkflowTemplate[];
}

export class WorkflowTemplateList extends BasePage<RouteComponentProps<any>, State> {
    private get search() {
        return this.queryParam('search') || '';
    }

    private set search(search) {
        this.setQueryParams({search});
    }

    constructor(props: any) {
        super(props);
        this.state = {};
    }

    public componentDidMount(): void {
        services.workflowTemplate.list('').then(templates => {
            this.setState({templates});
        });
    }

    public render() {
        return (
            <Page
                title='Workflow Templates'
                toolbar={{
                    breadcrumbs: [{title: 'Workflow Templates', path: uiUrl('workflow-templates')}]
                }}>
                <div className='argo-container'>{this.renderTemplates()}</div>
            </Page>
        );
    }

    private renderTemplates() {
        if (this.state.templates === undefined) {
            return <MockupList />;
        }
        if (this.state.templates.length === 0) {
            return (
                <div className='white-box'>
                    <h4>No workflow templates</h4>
                    <p>You can create new templates using the CLI.</p>
                </div>
            );
        }
        const templates = this.state.templates.filter(tmpl => tmpl.metadata.name.indexOf(this.search) >= 0);
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
                    />
                </p>
                {templates.length === 0 ? (
                    <p>No workflow templates found</p>
                ) : (
                    <div className={'argo-table-list'}>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-4'>NAME</div>
                            <div className='columns small-4'>NAMESPACE</div>
                            <div className='columns small-4'>CREATED</div>
                        </div>
                        {templates.map(t => (
                            <Link className='row argo-table-list__row' key={t.metadata.name} to={uiUrl(`workflow-templates/${t.metadata.namespace}/${t.metadata.name}`)}>
                                <div className='columns small-4'>{t.metadata.name}</div>
                                <div className='columns small-4'>{t.metadata.namespace}</div>
                                <div className='columns small-4'>
                                    <Timestamp date={t.metadata.creationTimestamp} />
                                </div>
                            </Link>
                        ))}
                    </div>
                )}
            </>
        );
    }
}

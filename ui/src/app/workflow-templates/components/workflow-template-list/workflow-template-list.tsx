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
        return this.getParam('search') || '';
    }

    private set search(search) {
        this.setParams({search});
    }

    private get templates() {
        return this.state.templates.filter(tmpl => tmpl.metadata.name.indexOf(this.search) >= 0);
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
                <div className='argo-container'>
                    <i className='fa fa-search' />
                    <input
                        className={'argo-field'}
                        defaultValue={this.search}
                        onChange={e => {
                            this.search = e.target.value;
                        }}
                    />
                    {this.renderTemplates()}
                </div>
            </Page>
        );
    }

    private renderTemplates() {
        if (this.state.templates === undefined) {
            return <MockupList />;
        }
        const templates = this.templates;
        if (templates.length === 0) {
            return <p>No workflow templates</p>;
        }
        return (
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
        );
    }
}

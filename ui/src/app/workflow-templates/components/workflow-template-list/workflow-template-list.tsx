import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {searchToMetadataFilter} from '../../../shared/filter';
import {services} from '../../../shared/services';
import {Consumer} from '../../../shared/context';
import {YamlEditor} from "../../../shared/components/yaml-editor/yaml-editor";
import { Utils } from '../../../shared/utils';

require('./workflow-template-list.scss');

const placeholderWorkflowTemplate: string = `apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  generateName: hello-world
spec:
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`;

interface State {
    templates?: models.WorkflowTemplate[];
    error?: Error;
}

export class WorkflowTemplateList extends BasePage<RouteComponentProps<any>, State> {
    private get search() {
        return this.queryParam('search') || '';
    }

    private get wfInput() {
        const query = new URLSearchParams(this.props.location.search);
        return Utils.tryJsonParse(query.get('new'));
    }

    private set search(search) {
        this.setQueryParams({search});
    }

    constructor(props: any) {
        super(props);
        this.state = {};
    }

    public componentDidMount(): void {
        services.workflowTemplate
            .list('')
            .then(templates => this.setState({templates}))
            .catch(error => this.setState({error}));
    }

    public render() {
        if (this.state.error) {
            throw this.state.error;
        }
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflow Templates'
                        toolbar={{
                            breadcrumbs: [{title: 'Workflow Templates', path: uiUrl('workflow-templates')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Create New Workflow Template',
                                        iconClassName: 'fa fa-plus',
                                        action: () => ctx.navigation.goto('.', {new: '{}'})
                                    }
                                ]
                            }
                        }}>
                        <div className='argo-container'>{this.renderTemplates()}</div>
                        <SlidingPanel isShown={!!this.wfInput} onClose={() => ctx.navigation.goto('.', {new: null})}>
                            Create Workflow Template
                            <YamlEditor
                                minHeight={800}
                                initialEditMode={true}
                                submitMode={true}
                                placeHolder={placeholderWorkflowTemplate}
                                onSave={rawWf => {
                                    // TODO(simon): Remove hardwired 'argo' namespace
                                    return services.workflowTemplate
                                        .create(JSON.parse(rawWf), 'argo')
                                        .then()
                                        .then(wf => ctx.navigation.goto(`/workflow-templates/${wf.metadata.namespace}/${wf.metadata.name}`));
                                }}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private renderTemplates() {
        if (!this.state.templates) {
            return <Loading/>;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/apiserverimpl/docs/workflow-templates.md'>Learn
            more</a>;
        if (this.state.templates.length === 0) {
            return (
                <div className='white-box'>
                    <h4>No workflow templates</h4>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>{learnMore}.</p>
                </div>
            );
        }
        const filter = searchToMetadataFilter(this.search);
        const templates = this.state.templates.filter(tmpl => filter(tmpl.metadata));
        return (
            <>
                <p>
                    <i className='fa fa-search'/>
                    <input
                        className='argo-field'
                        defaultValue={this.search}
                        onChange={e => {
                            this.search = e.target.value;
                        }}
                        placeholder='e.g. name:hello-world namespace:argo'
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
                            <Link className='row argo-table-list__row' key={t.metadata.name}
                                  to={uiUrl(`workflow-templates/${t.metadata.namespace}/${t.metadata.name}`)}>
                                <div className='columns small-4'>{t.metadata.name}</div>
                                <div className='columns small-4'>{t.metadata.namespace}</div>
                                <div className='columns small-4'>
                                    <Timestamp date={t.metadata.creationTimestamp}/>
                                </div>
                            </Link>
                        ))}
                    </div>
                )}
                <p>
                    <i className='fa fa-info-circle'/> Workflow templates are reusable templates you can create new
                    workflows from. {learnMore}.
                </p>
            </>
        );
    }
}

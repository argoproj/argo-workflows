import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {Timestamp} from '../../../shared/components/timestamp';
import {YamlEditor} from '../../../shared/components/yaml-editor/yaml-editor';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

require('./workflow-template-list.scss');

const placeholderWorkflowTemplate = (namespace: string) => `apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world
  namespace: ${namespace}
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
    private get namespace() {
        return this.queryParam('namespace') || '';
    }

    private set namespace(namespace) {
        this.setQueryParams({namespace});
    }

    private get wfInput() {
        return Utils.tryJsonParse(this.queryParam('new'));
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        services.workflowTemplate
            .list(this.namespace)
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
                            },
                            tools: [<NamespaceFilter key='namespace-filter' value={this.namespace} onChange={namespace => (this.namespace = namespace)} />]
                        }}>
                        {this.renderTemplates()}
                        <SlidingPanel isShown={!!this.wfInput} onClose={() => ctx.navigation.goto('.', {new: null})}>
                            Create Workflow Template
                            <YamlEditor
                                minHeight={800}
                                initialEditMode={true}
                                submitMode={true}
                                placeHolder={placeholderWorkflowTemplate(this.namespace || 'default')}
                                onSave={rawWf => {
                                    return services.workflowTemplate
                                        .create(JSON.parse(rawWf))
                                        .then(wf => ctx.navigation.goto(`/workflow-templates/${wf.metadata.namespace}/${wf.metadata.name}`))
                                        .catch(error => this.setState({error}));
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
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/apiserverimpl/docs/workflow-templates.md'>Learn more</a>;
        if (this.state.templates.length === 0) {
            return (
                <ZeroState title='No workflow templates'>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>{learnMore}.</p>
                </ZeroState>
            );
        }
        return (
            <div className='row'>
                <div className='columns small-12 xxlarge-2'>
                    <div className='argo-table-list'>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-1' />
                            <div className='columns small-5'>NAME</div>
                            <div className='columns small-3'>NAMESPACE</div>
                            <div className='columns small-3'>CREATED</div>
                        </div>
                        {this.state.templates.map(t => (
                            <Link
                                className='row argo-table-list__row'
                                key={`${t.metadata.namespace}/${t.metadata.name}`}
                                to={uiUrl(`workflow-templates/${t.metadata.namespace}/${t.metadata.name}`)}>
                                <div className='columns small-1'>
                                    <i className='fa fa-clone' />
                                </div>
                                <div className='columns small-5'>{t.metadata.name}</div>
                                <div className='columns small-3'>{t.metadata.namespace}</div>
                                <div className='columns small-3'>
                                    <Timestamp date={t.metadata.creationTimestamp} />
                                </div>
                            </Link>
                        ))}
                    </div>
                    <p>
                        <i className='fa fa-info-circle' /> Workflow templates are reusable templates you can create new workflows from. {learnMore}.
                    </p>
                </div>
            </div>
        );
    }
}

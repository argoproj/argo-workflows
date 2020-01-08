import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {YamlEditor} from '../../../shared/components/yaml-editor/yaml-editor';
import {Consumer} from '../../../shared/context';
import {searchToMetadataFilter} from '../../../shared/filter';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

require('./cron-workflow-list.scss');

const placeholderCronWorkflow: string = `apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  generateName: hello-world
spec:
  schedule: * * * * *
  workflowSpec:
      templates:
      - name: whalesay
        container:
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["hello world"]
`;

interface State {
    cronWorkflows?: models.CronWorkflow[];
    error?: Error;
}

export class CronWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private get search() {
        return this.queryParam('search') || '';
    }

    private set search(search) {
        this.setQueryParams({search});
    }

    private get wfInput() {
        const query = new URLSearchParams(this.props.location.search);
        return Utils.tryJsonParse(query.get('new'));
    }

    constructor(props: any) {
        super(props);
        this.state = {};
    }

    public componentDidMount(): void {
        services.cronWorkflows
            .list('')
            .then(cronWorkflows => this.setState({cronWorkflows}))
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
                        title='Cron Workflows'
                        toolbar={{
                            breadcrumbs: [{title: 'Cron Workflows', path: uiUrl('cron-workflows')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Create New Cron Workflow',
                                        iconClassName: 'fa fa-plus',
                                        action: () => ctx.navigation.goto('.', {new: '{}'})
                                    }
                                ]
                            }
                        }}>
                        <div className='argo-container'>{this.renderTemplates()}</div>
                        <SlidingPanel isShown={!!this.wfInput} onClose={() => ctx.navigation.goto('.', {new: null})}>
                            Create Cron Workflow
                            <YamlEditor
                                minHeight={800}
                                initialEditMode={true}
                                submitMode={true}
                                placeHolder={placeholderCronWorkflow}
                                onSave={rawWf => {
                                    // TODO(simon): Remove hardwired 'argo' namespace
                                    return services.cronWorkflows
                                        .create(JSON.parse(rawWf), 'argo')
                                        .then(cronWf => ctx.navigation.goto(`/cron-workflows/${cronWf.metadata.namespace}/${cronWf.metadata.name}`))
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
        if (!this.state.cronWorkflows) {
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/apiserverimpl/docs/cron-workflows.md'>Learn more</a>;
        if (this.state.cronWorkflows.length === 0) {
            return (
                <div className='white-box'>
                    <h4>No Cron Workflows</h4>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>{learnMore}.</p>
                </div>
            );
        }
        const filter = searchToMetadataFilter(this.search);
        const cronWorkflows = this.state.cronWorkflows.filter(tmpl => filter(tmpl.metadata));
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
                {cronWorkflows.length === 0 ? (
                    <p>No cron workflows found</p>
                ) : (
                    <div className={'argo-table-list'}>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-4'>NAME</div>
                            <div className='columns small-4'>NAMESPACE</div>
                            <div className='columns small-4'>CREATED</div>
                        </div>
                        {cronWorkflows.map(cronWf => (
                            <Link
                                className='row argo-table-list__row'
                                key={cronWf.metadata.name}
                                to={uiUrl(`workflow-templates/${cronWf.metadata.namespace}/${cronWf.metadata.name}`)}>
                                <div className='columns small-4'>{cronWf.metadata.name}</div>
                                <div className='columns small-4'>{cronWf.metadata.namespace}</div>
                                <div className='columns small-4'>
                                    <Timestamp date={cronWf.metadata.creationTimestamp} />
                                </div>
                            </Link>
                        ))}
                    </div>
                )}
                <p>
                    <i className='fa fa-info-circle' /> Cron Workflows are Workflows that run on a preset schedule. {learnMore}.
                </p>
            </>
        );
    }
}

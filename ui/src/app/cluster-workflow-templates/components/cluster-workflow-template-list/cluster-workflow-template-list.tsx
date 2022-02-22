import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {Footnote} from '../../../shared/footnote';
import {services} from '../../../shared/services';
import {ClusterWorkflowTemplateCreator} from '../cluster-workflow-template-creator';

require('./cluster-workflow-template-list.scss');

interface State {
    templates?: models.ClusterWorkflowTemplate[];
    error?: Error;
}

export class ClusterWorkflowTemplateList extends BasePage<RouteComponentProps<any>, State> {
    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        this.fetchClusterWorkflowTemplates();
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Cluster Workflow Templates'
                        toolbar={{
                            breadcrumbs: [{title: 'Cluster Workflow Templates', path: uiUrl('cluster-workflow-templates')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Create New Cluster Workflow Template',
                                        iconClassName: 'fa fa-plus',
                                        action: () => (this.sidePanel = 'new')
                                    }
                                ]
                            }
                        }}>
                        {this.renderTemplates()}
                        <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                            <ClusterWorkflowTemplateCreator onCreate={wf => ctx.navigation.goto(uiUrl(`cluster-workflow-templates/${wf.metadata.name}`))} />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchClusterWorkflowTemplates(): void {
        services.clusterWorkflowTemplate
            .list()
            .then(templates => this.setState({error: null, templates}))
            .catch(error => this.setState({error}));
    }

    private renderTemplates() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.templates) {
            return <Loading />;
        }
        const learnMore = <a href='https://argoproj.github.io/argo-workflows/cluster-workflow-templates/'>Learn more</a>;
        if (this.state.templates.length === 0) {
            return (
                <ZeroState title='No cluster workflow templates'>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>
                        <ExampleManifests />. {learnMore}.
                    </p>
                </ZeroState>
            );
        }
        return (
            <>
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1' />
                        <div className='columns small-5'>NAME</div>
                        <div className='columns small-3'>CREATED</div>
                    </div>
                    {this.state.templates.map(t => (
                        <Link className='row argo-table-list__row' key={t.metadata.uid} to={uiUrl(`cluster-workflow-templates/${t.metadata.name}`)}>
                            <div className='columns small-1'>
                                <i className='fa fa-clone' />
                            </div>
                            <div className='columns small-5'>{t.metadata.name}</div>
                            <div className='columns small-3'>
                                <Timestamp date={t.metadata.creationTimestamp} />
                            </div>
                        </Link>
                    ))}
                </div>
                <Footnote>
                    <InfoIcon /> Cluster scoped Workflow templates are reusable templates you can create new workflows from. <ExampleManifests />. {learnMore}.
                </Footnote>
            </>
        );
    }
}

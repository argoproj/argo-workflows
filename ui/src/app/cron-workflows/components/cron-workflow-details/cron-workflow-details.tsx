import {NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {CronWorkflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {services} from '../../../shared/services';
import {CronWorkflowSummaryPanel} from '../cron-workflow-summary-panel';

const jsonMergePatch = require('json-merge-patch');
require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    cronWorkflow?: CronWorkflow;
    error?: Error;
}

export class CronWorkflowDetails extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.props.match.params.namespace;
    }

    private get name() {
        return this.props.match.params.name;
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        services.cronWorkflows
            .get(this.name, this.namespace)
            .then(cronWf => this.setState({cronWorkflow: cronWf}))
            .catch(error => this.setState({error}));
    }

    public render() {
        if (this.state.error !== undefined) {
            throw this.state.error;
        }
        return (
            <Page
                title='Cron Workflow Details'
                toolbar={{
                    actionMenu: {
                        items: [
                            {
                                title: 'Delete',
                                iconClassName: 'fa fa-trash',
                                action: () => this.deleteWorkflowTemplate()
                            },
                            {
                                title: 'Suspend',
                                iconClassName: 'fa fa-pause',
                                action: () => this.suspendCronWorkflow(),
                                disabled: !this.state.cronWorkflow || this.state.cronWorkflow.spec.suspend
                            },
                            {
                                title: 'Resume',
                                iconClassName: 'fa fa-play',
                                action: () => this.resumeCronWorkflow(),
                                disabled: !this.state.cronWorkflow || !this.state.cronWorkflow.spec.suspend
                            }
                        ]
                    },
                    breadcrumbs: [
                        {
                            title: 'Cron Workflows',
                            path: uiUrl('cron-workflows')
                        },
                        {title: this.namespace + '/' + this.name}
                    ]
                }}>
                <div className='argo-container'>
                    <div className='workflow-details__content'>{this.renderCronWorkflow()}</div>
                </div>
            </Page>
        );
    }

    private renderCronWorkflow() {
        if (!this.state.cronWorkflow) {
            return <Loading />;
        }
        return (
            <CronWorkflowSummaryPanel cronWorkflow={this.state.cronWorkflow} onChange={cronWorkflow => this.setState({cronWorkflow})} onError={error => this.setState({error})} />
        );
    }

    private deleteWorkflowTemplate() {
        if (!confirm('Are you sure you want to delete this cron workflow?\nThere is no undo.')) {
            return;
        }
        services.cronWorkflows
            .delete(this.name, this.namespace)
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to delete cron workflow ' + e,
                    type: NotificationType.Error
                });
            })
            .then(() => {
                document.location.href = uiUrl('cron-workflows');
            });
    }

    private suspendCronWorkflow() {
        const wf = JSON.parse(JSON.stringify(this.state.cronWorkflow));
        wf.spec.suspend = true;
        const patch = jsonMergePatch.generate(this.state.cronWorkflow, wf) || {};
        services.cronWorkflows
            .get(this.name, this.namespace)
            .then(latest => jsonMergePatch.apply(latest, patch))
            .then(patched => services.cronWorkflows
                .update(patched, this.name, this.namespace)
            )
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to suspend cron workflow ' + e,
                    type: NotificationType.Error
                });
            })
            .then((updated: CronWorkflow) => this.setState({ cronWorkflow: updated }));
    }

    private resumeCronWorkflow() {
        const wf = JSON.parse(JSON.stringify(this.state.cronWorkflow));
        wf.spec.suspend = undefined;
        const patch = jsonMergePatch.generate(this.state.cronWorkflow, wf) || {};
        services.cronWorkflows
            .get(this.name, this.namespace)
            .then(latest => jsonMergePatch.apply(latest, patch))
            .then(patched => services.cronWorkflows
                .update(patched, this.name, this.namespace)
            )
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to resume cron workflow ' + e,
                    type: NotificationType.Error
                });
            })
            .then((updated: CronWorkflow) => this.setState({ cronWorkflow: updated }));
    }
}

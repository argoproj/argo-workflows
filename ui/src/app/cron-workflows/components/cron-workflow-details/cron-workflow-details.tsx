import {NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {CronWorkflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {services} from '../../../shared/services';
import {CronWorkflowSummaryPanel} from '../cron-workflow-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    cronWorkflow?: CronWorkflow;
    error?: Error;
}

export class CronWorkflowDetails extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.props.match.params.namespace || '';
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
            .then(cronWf => this.setState({error: null, cronWorkflow: cronWf}))
            .catch(error => this.setState({error}));
    }

    public render() {
        const suspendButton =
            this.state.cronWorkflow && !this.state.cronWorkflow.spec.suspend
                ? {
                      title: 'Suspend',
                      iconClassName: 'fa fa-pause',
                      action: () => this.suspendCronWorkflow(),
                      disabled: !this.state.cronWorkflow
                  }
                : {
                      title: 'Resume',
                      iconClassName: 'fa fa-play',
                      action: () => this.resumeCronWorkflow(),
                      disabled: !this.state.cronWorkflow || !this.state.cronWorkflow.spec.suspend
                  };
        return (
            <Page
                title='Cron Workflow Details'
                toolbar={{
                    actionMenu: {
                        items: [
                            {
                                title: 'Submit',
                                iconClassName: 'fa fa-plus',
                                action: () => this.submitCronWorkflow()
                            },
                            {
                                title: 'Delete',
                                iconClassName: 'fa fa-trash',
                                action: () => this.deleteCronWorkflow()
                            },
                            suspendButton
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
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.cronWorkflow) {
            return <Loading />;
        }
        return <CronWorkflowSummaryPanel cronWorkflow={this.state.cronWorkflow} onChange={cronWorkflow => this.setState({cronWorkflow})} />;
    }

    private async submitCronWorkflow() {
        try {
            const submitted = await services.workflows.submit('cronwf', this.name, this.namespace);

            try {
                document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`);
            } catch (e) {
                this.appContext.apis.notifications.show({
                    content: 'Failed redirect to newly submitted cron workflow ' + e,
                    type: NotificationType.Error
                });
            }
        } catch (e) {
            this.appContext.apis.notifications.show({
                content: 'Failed to submit cron workflow ' + e,
                type: NotificationType.Error
            });
        }
    }

    private deleteCronWorkflow() {
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
        services.cronWorkflows
            .suspend(this.name, this.namespace)
            .then((updated: CronWorkflow) => this.setState({cronWorkflow: updated}))
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to suspend cron workflow ' + e,
                    type: NotificationType.Error
                });
            });
    }

    private resumeCronWorkflow() {
        services.cronWorkflows
            .resume(this.name, this.namespace)
            .then((updated: CronWorkflow) => this.setState({cronWorkflow: updated}))
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to resume cron workflow ' + e,
                    type: NotificationType.Error
                });
            });
    }
}

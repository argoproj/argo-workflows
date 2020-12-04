import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {EventSourceSummaryPanel} from "../event-source-summary-panel";
import {EventSource} from "../../../../models";


require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    namespace?: string;
    eventSource?: EventSource;
    error?: Error;
}

export class EventSourceDetails extends BasePage<RouteComponentProps<any>, State> {
    private get name() {
        return this.props.match.params.name;
    }

    private get namespace(){
        return this.props.match.params.namespace;
    }

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
        services.eventSource
            .get(this.name, this.namespace)
            .then(eventSource => this.setState({error: null, eventSource: eventSource}))
            .then(() => services.info.getInfo())
            .then(info => this.setState({namespace: info.managedNamespace || Utils.getCurrentNamespace() || 'default'}))
            .catch(error => this.setState({error}));
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Event Source Details'
                        toolbar={{
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Delete',
                                        iconClassName: 'fa fa-trash',
                                        action: () => this.deleteEventSource()
                                    }
                                ]
                            },
                            breadcrumbs: [
                                {
                                    title: 'Event Source',
                                    path: uiUrl('event-sources')
                                },
                                {title: this.name}
                            ]
                        }}>
                        <div className='argo-container'>
                            <div className='workflow-details__content'>{this.renderEventSource()}</div>
                        </div>
                        {this.state.eventSource && (
                            <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                                {/*<SubmitWorkflowPanel*/}
                                {/*    kind='ClusterWorkflowTemplate'*/}
                                {/*    namespace={this.state.namespace}*/}
                                {/*    name={this.state.eventSource.metadata.name}*/}
                                {/*/>*/}
                            </SlidingPanel>
                        )}
                    </Page>
                )}
            </Consumer>
        );
    }

    private renderEventSource() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.eventSource) {
            return <Loading />;
        }
        return <EventSourceSummaryPanel eventSource={this.state.eventSource} onChange={eventSource => this.setState({eventSource: eventSource})} />;
    }

    private deleteEventSource() {
        if (!confirm('Are you sure you want to delete this event source?\nThere is no undo.')) {
            return;
        }
        services.eventSource
            .delete(this.name, this.state.namespace)
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to delete event source' + e,
                    type: NotificationType.Error
                });
            })
            .then(() => {
                document.location.href = uiUrl('event-sources');
            });
    }
}

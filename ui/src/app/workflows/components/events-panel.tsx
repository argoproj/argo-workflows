import * as React from 'react';
import {Subscription} from 'rxjs';
import {Event} from '../../../models';
import {ErrorNotice} from '../../shared/components/error-notice';
import {Notice} from '../../shared/components/notice';
import {Timestamp} from '../../shared/components/timestamp';
import {ToggleButton} from '../../shared/components/toggle-button';
import {services} from '../../shared/services';

interface Props {
    namespace: string;
    name: string;
    kind: string;
}

interface State {
    showAll: boolean;
    hideNormal: boolean;
    events?: Event[];
    error?: Error;
}

export class EventsPanel extends React.Component<Props, State> {
    private get fieldSelector() {
        const fieldSelectors: string[] = [];
        if (!this.showAll) {
            fieldSelectors.push('involvedObject.kind=' + this.props.kind);
            fieldSelectors.push('involvedObject.name=' + this.props.name);
        }
        if (this.hideNormal) {
            fieldSelectors.push('type!=Normal');
        }
        return fieldSelectors.join(',');
    }

    private set showAll(showAll: boolean) {
        this.setState({showAll, events: undefined}, () => this.fetchEvents());
    }

    private get showAll() {
        return this.state.showAll;
    }

    private set hideNormal(hideNormal: boolean) {
        this.setState({hideNormal, events: undefined}, () => this.fetchEvents());
    }

    private get hideNormal() {
        return this.state.hideNormal;
    }

    private subscription?: Subscription;

    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {showAll: false, hideNormal: false};
    }

    public componentDidMount() {
        this.fetchEvents();
    }

    public componentWillUnmount() {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
    }

    public render() {
        return (
            <>
                <div style={{margin: 20}}>
                    <ToggleButton toggled={this.showAll} onToggle={() => (this.showAll = !this.showAll)} title='Show all events in the namespace'>
                        Show All
                    </ToggleButton>
                    <ToggleButton toggled={this.hideNormal} onToggle={() => (this.hideNormal = !this.hideNormal)} title='Hide normal events'>
                        Hide normal
                    </ToggleButton>
                </div>
                {this.renderEvents()}
            </>
        );
    }

    private fetchEvents() {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
        this.subscription = services.workflows
            .watchEvents(this.props.namespace, this.fieldSelector)
            .map(event => {
                const events = this.state.events || [];
                const index = events.findIndex(item => item.metadata.uid === event.metadata.uid);
                if (index > -1 && event.metadata.resourceVersion === events[index].metadata.resourceVersion) {
                    return events;
                }
                if (index > -1) {
                    events[index] = event;
                } else {
                    events.unshift(event);
                }
                return events;
            })
            .subscribe(
                events => this.setState({events}),
                error => this.setState({error})
            );
    }

    private renderEvents() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} style={{margin: 20}} />;
        }
        if (!this.state.events || this.state.events.length === 0) {
            return (
                <Notice style={{margin: 20}}>
                    <i className='fa fa-spin fa-circle-notch' /> Waiting for events. Still waiting for data? Try changing the filters.
                </Notice>
            );
        }
        return (
            <div className='argo-table-list'>
                <div className='row argo-table-list__head'>
                    <div className='columns small-1'>Type</div>
                    <div className='columns small-2'>Last Seen</div>
                    <div className='columns small-2'>Reason</div>
                    <div className='columns small-2'>Object</div>
                    <div className='columns small-5'>Message</div>
                </div>
                {this.state.events.map(e => (
                    <div className='row argo-table-list__row' key={e.metadata.uid}>
                        <div className='columns small-1' title={e.type}>
                            {e.type === 'Normal' ? <i className='fa fa-check-circle status-icon--init' /> : <i className='fa fa-exclamation-circle status-icon--pending' />}
                        </div>
                        <div className='columns small-2'>
                            <Timestamp date={e.lastTimestamp} />
                        </div>
                        <div className='columns small-2'>{e.reason}</div>

                        <div className='columns small-2'>
                            {e.involvedObject.kind}/{e.involvedObject.name}
                        </div>
                        <div className='columns small-5'>{e.message}</div>
                    </div>
                ))}
            </div>
        );
    }
}

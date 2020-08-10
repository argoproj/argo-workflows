import * as React from 'react';
import {Subscription} from 'rxjs';
import {Event} from '../../../models';
import {ErrorNotice} from '../../shared/components/error-notice';
import {Loading} from '../../shared/components/loading';
import {PhaseIcon} from '../../shared/components/phase-icon';
import {Timestamp} from '../../shared/components/timestamp';
import {services} from '../../shared/services';

interface Props {
    namespace: string;
    fieldSelector: string;
}

interface State {
    events?: Event[];
    error?: Error;
}

export class EventsPanel extends React.Component<Props, State> {
    private subscription: Subscription;

    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {};
    }

    public componentDidMount() {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
        this.subscription = services.workflows
            .watchEvents(this.props.namespace, this.props.fieldSelector)
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

    public componentWillUnmount() {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
    }

    public render() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} style={{margin: 20}} />;
        }
        if (!this.state.events) {
            return <Loading />;
        }
        return (
            <div className='argo-table-list'>
                <div className='row argo-table-list__head'>
                    <div className='columns small-1'>Type</div>
                    <div className='columns small-2'>Last Seen</div>
                    <div className='columns small-3'>Reason</div>
                    <div className='columns small-6'>Message</div>
                </div>
                {this.state.events.map(e => (
                    <div className='row argo-table-list__row' key={e.metadata.uid}>
                        <div className='columns small-1' title={e.type}>
                            <PhaseIcon value={e.type === 'Normal' ? 'Succeeded' : 'Failed'} />
                        </div>
                        <div className='columns small-2'>
                            <Timestamp date={e.lastTimestamp} />
                        </div>
                        <div className='columns small-3'>{e.reason}</div>
                        <div className='columns small-6'>{e.message}</div>
                    </div>
                ))}
            </div>
        );
    }
}

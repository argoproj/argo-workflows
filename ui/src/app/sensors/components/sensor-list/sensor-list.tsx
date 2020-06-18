import {Page} from 'argo-ui';

import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Sensor} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {ZeroState} from '../../../shared/components/zero-state';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

interface State {
    initialized: boolean;
    managedNamespace: boolean;
    namespace: string;
    sensors?: Sensor[];
    error?: Error;
}

export class SensorList extends BasePage<RouteComponentProps<any>, State> {
    private static renderPhase(phase: string) {
        switch (phase) {
            case 'Complete':
                return <i className='fa fa-check-circle status-icon--success' />;
            case 'Active':
                return <i className='fa fa-circle-notch fa-spin status-icon--running' />;
            case 'Error':
                return <i className='fa fa-times-circle status-icon--failed' />;
            default:
                return <i className='fa fa-clock status-icon--init' />;
        }
    }
    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {
            initialized: false,
            managedNamespace: false,
            namespace: this.props.match.params.namespace || Utils.getCurrentNamespace() || ''
        };
    }

    public componentDidMount(): void {
        this.fetchSensors(this.state.namespace);
    }

    public render() {
        if (this.state.error) {
            throw this.state.error;
        }
        if (!this.state.sensors) {
            return <Loading />;
        }
        return (
            <Page
                title='Sensors'
                toolbar={{
                    breadcrumbs: [{title: 'Sensors', path: uiUrl('sensors')}],
                    actionMenu: {
                        items: []
                    },
                    tools: []
                }}>
                {this.renderSensors()}
            </Page>
        );
    }

    private fetchSensors(namespace: string): void {
        let sensorList;
        let newNamespace = namespace;
        if (!this.state.initialized) {
            sensorList = services.info.getInfo().then(info => {
                if (info.managedNamespace) {
                    newNamespace = info.managedNamespace;
                }
                this.setState({initialized: true, managedNamespace: !!info.managedNamespace});
                return services.sensors.list(newNamespace);
            });
        } else {
            if (this.state.managedNamespace) {
                newNamespace = this.state.namespace;
            }
            sensorList = services.sensors.list(newNamespace);
        }
        sensorList
            .then(list => {
                this.setState({
                    namespace: newNamespace,
                    sensors: list.items || []
                });
                Utils.setCurrentNamespace(newNamespace);
            })
            .catch(error => this.setState({error}));
    }

    private renderSensors() {
        if (!this.state.sensors) {
            return <Loading />;
        }
        if (this.state.sensors.length === 0) {
            return (
                <ZeroState title='No senors'>
                    <p>No sensors set-up.</p>
                </ZeroState>
            );
        }

        return (
            <>
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1' />
                        <div className='columns small-2'>NAME</div>
                        <div className='columns small-2'>NAMESPACE</div>
                        <div className='columns small-3'>MESSAGE</div>
                        <div className='columns small-2'>DEPENDENCIES</div>
                        <div className='columns small-2'>TRIGGERS</div>
                    </div>
                    {this.state.sensors.map(s => (
                        <Link className='row argo-table-list__row' key={`${s.metadata.uid}`} to={uiUrl(`sensors/${s.metadata.namespace}/${s.metadata.name}`)}>
                            <div className='columns small-1'>{SensorList.renderPhase(s.status.phase)}</div>
                            <div className='columns small-2'>{s.metadata.name}</div>
                            <div className='columns small-2'>{s.metadata.namespace}</div>
                            <div className='columns small-3'>{s.status.message || '-'}</div>
                            <div className='columns small-2'>{s.spec.dependencies.map(d => d.name).join(',')}</div>
                            <div className='columns small-2'>
                                {s.spec.triggers
                                    .map(t => t.template)
                                    .map(t => t.name)
                                    .join(',')}
                            </div>
                        </Link>
                    ))}
                </div>
            </>
        );
    }
}

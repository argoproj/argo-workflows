import {Page} from 'argo-ui';

import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {Sensor} from '../../model/sensors';
import {PhaseIcon} from '../shared/phases-icon';
import {SensorFilters} from './sensor-filters';

interface State {
    initialized: boolean;
    managedNamespace: boolean;
    namespace: string;
    sensors?: Sensor[];
    error?: Error;
}

export class SensorList extends BasePage<RouteComponentProps<any>, State> {
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
                <div className='row'>
                    <div className='columns small-12 xlarge-2'>
                        <SensorFilters namespace={this.state.namespace} onChange={namespace => this.changeFilters(namespace)} />
                    </div>
                    <div className='columns small-12 xlarge-10'>{this.renderSensors()}</div>
                </div>
            </Page>
        );
    }

    private fetchSensors(namespace: string) {
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
                    <p>No sensors found</p>
                    <p><a href='https://argoproj.github.io/argo-events/'>Learn more</a></p>
                </ZeroState>
            );
        }
        return (
            <>
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1' />
                        <div className='columns small-3'>Name</div>
                        <div className='columns small-3'>Namespace</div>
                        <div className='columns small-2'>Started At</div>
                        <div className='columns small-2'>Completed At</div>
                    </div>
                    {this.state.sensors.map(s => (
                        <Link className='row argo-table-list__row' key={`${s.metadata.uid}`} to={uiUrl(`sensors/${s.metadata.namespace}/${s.metadata.name}`)}>
                            <div className='columns small-1'>
                                <PhaseIcon value={s.status.phase} />
                            </div>
                            <div className='columns small-3'>{s.metadata.name}</div>
                            <div className='columns small-3'>{s.metadata.namespace}</div>
                            <div className='columns small-2'>
                                <Timestamp date={s.status.startedAt} />
                            </div>
                            <div className='columns small-2'>
                                <Timestamp date={s.status.completedAt} />
                            </div>
                        </Link>
                    ))}
                </div>
            </>
        );
    }

    private changeFilters(namespace: string) {
        this.setState({namespace});
        history.pushState(null, '', uiUrl('sensors/' + namespace));
        this.fetchSensors(namespace);
    }
}

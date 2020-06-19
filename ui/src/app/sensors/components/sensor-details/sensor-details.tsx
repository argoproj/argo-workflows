import {Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {services} from '../../../shared/services';
import {NodeStatus, Sensor, Trigger} from '../../model/sensors';
import {PhaseIcon} from '../shared/phases-icon';
import {TriggerCycleStatusIcon} from '../shared/trigger-cycle-status-icon';
import {YamlViewer} from "../../../shared/components/yaml/yaml-viewer";
import * as jsYaml from "js-yaml";
import {TriggerParametersPanel} from "../shared/trigger-parameters-panel";

interface State {
    sensor?: Sensor;
    error?: Error;
}

export class SensorDetails extends BasePage<RouteComponentProps<any>, State> {
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
        services.sensors
            .get(this.name, this.namespace)
            .then(sensor => this.setState({sensor}))
            .catch(error => this.setState({error}));
    }

    public render() {
        if (this.state.error !== undefined) {
            throw this.state.error;
        }
        return (
            <Page
                title='Sensor Details'
                toolbar={{
                    actionMenu: {
                        items: []
                    },
                    breadcrumbs: [
                        {
                            title: 'Sensors',
                            path: uiUrl('sensors')
                        },
                        {title: this.namespace + '/' + this.name}
                    ]
                }}>
                {this.renderSensor()}
            </Page>
        );
    }

    private renderSensor() {
        if (!this.state.sensor) {
            return <Loading/>;
        }
        return (
            <div className='argo-container'>
                {this.renderSummary()}
                {this.renderTriggerCycleStatus()}
                {this.renderNodes()}
                {this.renderTriggers()}
                <br/>
                <div className='white-box'>
                    <YamlViewer value={jsYaml.dump(this.state.sensor)}/>
                </div>
            </div>
        );
    }

    private renderSummary() {
        return <div>
            <h6>Summary</h6>
            <div className='row argo-table-list__head'>
                <div className='columns small-2'/>
                <div className='columns small-2'>Name</div>
                <div className='columns small-2'>Namespace</div>
                <div className='columns small-2'>Started At</div>
                <div className='columns small-2'>Completed At</div>
                <div className='columns small-2'/>
            </div>
            <div className='row argo-table-list__row'>
                <div className='columns small-2'>
                    <PhaseIcon value={this.state.sensor.status.phase}/> {this.state.sensor.status.phase}
                </div>
                <div className='columns small-2'>{this.state.sensor.metadata.name}</div>
                <div className='columns small-2'>{this.state.sensor.metadata.namespace}</div>
                <div className='columns small-2'>
                    <Timestamp date={this.state.sensor.status.startedAt}/>
                </div>
                <div className='columns small-2'>
                    <Timestamp date={this.state.sensor.status.completedAt}/>
                </div>
                <div className='columns small-2'>{this.state.sensor.status.message || '-'}</div>
            </div>
        </div>;
    }

    private renderTriggers() {
        return <div>
            <h6>Triggers</h6>
            {this.state.sensor.spec.triggers.map(trigger => SensorDetails.renderTrigger(trigger))}
        </div>;
    }

    private renderNodes() {
        return <>
            <h6>Dependencies</h6>
            <div>
                <div className='row argo-table-list__head'>
                    <div className='columns small-1'/>
                    <div className='columns small-2'>Name</div>
                    <div className='columns small-2'>Type</div>
                    <div className='columns small-2'>Started At</div>
                    <div className='columns small-2'>Completed At</div>
                    <div className='columns small-3'/>
                </div>
                {Object.entries(this.state.sensor.status.nodes).map(([key,node]) => SensorDetails.renderNode(node))}
            </div>
        </>;
    }

    private renderTriggerCycleStatus() {
        return <>
            <h6>Trigger Cycle</h6>
            <div>
                <div className='row argo-table-list__head'>
                    <div className='columns small-2'/>
                    <div className='columns small-2'/>
                    <div className='columns small-2'>Count</div>
                </div>
                <div className='row argo-table-list__row'>
                    <div className='columns small-2'>
                        <TriggerCycleStatusIcon
                            value={this.state.sensor.status.triggerCycleStatus || ''}/> {this.state.sensor.status.triggerCycleStatus}
                    </div>
                    <div className='columns small-2'>
                        <Timestamp date={this.state.sensor.status.lastCycleTime}/>
                    </div>
                    <div className='columns small-2'>{this.state.sensor.status.triggerCycleCount}</div>
                </div>
            </div>
        </>;
    }

    private static renderNode(node: NodeStatus) {
        return <div className='row argo-table-list__row' key={node.name}>
            <div className='columns small-1'>
                <PhaseIcon value={node.phase}/>
            </div>
            <div className='columns small-2'>{node.displayName}</div>
            <div className='columns small-2'>{node.type}</div>
            <div className='columns small-2'>
                <Timestamp date={node.startedAt}/>
            </div>
            <div className='columns small-2'>
                <Timestamp date={node.completedAt}/>
            </div>
            <div className='columns small-3'>{node.message}</div>
            {node.event && (
                <>
                    <div className='columns small-1'/>
                    <div className='columns small-11'>
                        <pre>{atob(node.event.data)}</pre>
                    </div>
                </>
            )}
        </div>;
    }

    private static renderTrigger(trigger: Trigger) {
        return <React.Fragment  key={trigger.template.name}>
            <div className='row argo-table-list__head' key={trigger.template.name+".header"}>
                <div className='columns small-12'>{trigger.template.name}</div>
            </div>
            {trigger.template.k8s &&
            <div className='row argo-table-list__row' key={trigger.template.name + ".k8s"}>
                <div className='columns small-1'>K8S</div>
                <div className='columns small-1'>{trigger.template.k8s.operation}</div>
                <div className='columns small-2'>{trigger.template.k8s.resource}</div>
                <div className='columns small-7 '><TriggerParametersPanel
                    parameters={trigger.template.k8s.parameters}/></div>
                <div className='columns small-1'>
                    <a href='https://argoproj.github.io/argo-events/triggers/k8s-object-trigger/'><i className='fa fa-question-circle'/></a>
                </div>
            </div>}
            {trigger.template.argoWorkflow &&
            <div className='row argo-table-list__row' key={trigger.template.name + ".argoWorkflow"}>
                <div className='columns small-1'>Argo Workflow</div>
                <div className='columns small-1'>{trigger.template.argoWorkflow.operation}</div>
                <div className='columns small-2'/>
                <div className='columns small-7'><TriggerParametersPanel
                    parameters={trigger.template.argoWorkflow.parameters}/></div>
                <div className='columns small-1'>
                    <a href='https://argoproj.github.io/argo-events/triggers/argo-workflow/'><i className='fa fa-question-circle'/></a>
                </div>
            </div>}
        </React.Fragment>;
    }
}

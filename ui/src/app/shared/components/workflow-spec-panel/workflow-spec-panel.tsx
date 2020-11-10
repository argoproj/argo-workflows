import {SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {GraphPanel} from '../graph/graph-panel';
import {ResourceEditor} from '../resource-editor/resource-editor';
import {types} from './types';
import {workflowSpecGraph} from './workflow-spec-graph';

export class WorkflowSpecPanel extends React.Component<{spec: WorkflowSpec}, {selectedId?: string}> {
    private set selectedId(selectedId: string) {
        this.setState({selectedId});
    }

    private get selectedId() {
        return this.state.selectedId;
    }

    private get selected(): {kind: string; value: any} {
        if (!this.selectedId) {
            return;
        }
        const parts = this.selectedId.split('/');
        if (parts.length < 1) {
            return {kind: 'WorkflowSpec', value: this.props.spec};
        }
        switch (parts[0]) {
            case 'Artifacts':
            case 'Parameters':
                return {kind: 'Arguments', value: this.props.spec.arguments};
            case 'OnExit':
                return {kind: 'string', value: this.props.spec.onExit};
            case 'Template':
                break;
            default:
                return {kind: 'WorkflowSpec', value: this.props.spec};
        }
        const template = this.props.spec.templates.find(t => t.name === parts[1]);
        if (template.dag && parts.length >= 3) {
            const task = template.dag.tasks.find(x => x.name === parts[2]);
            if (parts.length === 4) {
                switch (parts[3]) {
                    case 'withItems':
                        return {kind: 'Items', value: task.withItems};
                    case 'withSequence':
                        return {kind: 'Sequence', value: task.withSequence};
                    case 'withParam':
                        return {kind: 'string', value: task.withParam};
                }
            }
            return {kind: 'DagTask', value: task};
        }
        if (template.steps && parts.length >= 4) {
            const step = template.steps[parseInt(parts[2], 10)][parseInt(parts[3], 10)];
            if (parts.length === 5) {
                switch (parts[4]) {
                    case 'withItems':
                        return {kind: 'Items', value: step.withItems};
                    case 'withSequence':
                        return {kind: 'Sequence', value: step.withSequence};
                    case 'withParam':
                        return {kind: 'string', value: step.withParam};
                }
            }
            return {kind: 'WorkflowStep', value: step};
        }
        return {kind: 'Template', value: template};
    }

    constructor(props: Readonly<{spec: WorkflowSpec}>) {
        super(props);
        this.state = {};
    }

    public render() {
        return (
            <div>
                <GraphPanel
                    storageKey='workflow-spec-panel'
                    graph={workflowSpecGraph(this.props.spec)}
                    selectedNode={this.selectedId}
                    onNodeSelect={id => (this.selectedId = id)}
                    horizontal={true}
                    types={types}
                    classNames={{'': true}}
                />
                <SlidingPanel key='template-editor' isShown={!!this.selected} onClose={() => (this.selectedId = null)}>
                    {this.selected && <ResourceEditor kind={this.selected.kind} title={this.selectedId} value={this.selected.value} />}
                </SlidingPanel>
            </div>
        );
    }
}

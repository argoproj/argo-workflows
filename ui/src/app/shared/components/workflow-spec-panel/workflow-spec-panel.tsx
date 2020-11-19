import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {GraphPanel} from '../graph/graph-panel';
import {types} from './types';
import {workflowSpecGraph} from './workflow-spec-graph';

export const WorkflowSpecPanel = (props: {spec: WorkflowSpec; selectedId?: string; onSelect?: (id: string) => void}) => {
    return (
        <GraphPanel
            graph={workflowSpecGraph(props.spec)}
            selectedNode={props.selectedId}
            onNodeSelect={id => props.onSelect && props.onSelect(id)}
            horizontal={true}
            nodeTypes={types}
            nodeClassNames={{'': true}}
            iconShapes={{
                when: 'circle',
                withItems: 'circle',
                withParam: 'circle',
                withSequence: 'circle',
                container: 'circle',
                script: 'circle',
                resource: 'circle'
            }}
        />
    );
};

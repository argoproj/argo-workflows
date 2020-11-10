import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {GraphPanel} from '../graph/graph-panel';
import {types} from './types';
import {workflowSpecGraph} from './workflow-spec-graph';

export const WorkflowSpecPanel = (props: {spec: WorkflowSpec; selectedId?: string; onSelect?: (id: string) => void}) => {
    return (
        <GraphPanel
            storageKey='workflow-spec-panel'
            graph={workflowSpecGraph(props.spec)}
            selectedNode={props.selectedId}
            onNodeSelect={id => props.onSelect(id)}
            horizontal={true}
            types={types}
            classNames={{'': true}}
        />
    );
};

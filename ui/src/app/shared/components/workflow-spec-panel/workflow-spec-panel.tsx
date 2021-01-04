import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {GraphPanel} from '../graph/graph-panel';
import {genres} from './genres';
import {workflowSpecGraph} from './workflow-spec-graph';

export const WorkflowSpecPanel = ({spec, selectedId, onSelect}: {spec: WorkflowSpec; selectedId?: string; onSelect?: (id: string) => void}) => {
    return (
        <GraphPanel
            storageScope='workflow-spec'
            graph={workflowSpecGraph(spec)}
            selectedNode={selectedId}
            onNodeSelect={id => onSelect && onSelect(id)}
            horizontal={true}
            nodeGenres={genres}
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

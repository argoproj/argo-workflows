import {SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {Resource} from '../../../../models';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph} from '../../../shared/components/graph/types';
import {ObjectEditor} from '../../../shared/components/object-editor/object-editor';
import {ListWatch} from '../../../shared/list-watch';
import {services} from '../../../shared/services';
import {icons} from './icons';

export const WorkflowResourcesGraph = ({namespace, name}: {namespace: string; name: string}) => {
    const [resources, setResources] = useState<Resource[]>();
    const [selectedNode, setSelectedNode] = useState<string>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        const lw = new ListWatch<Resource>(
            // no list function, so we fake it
            () => Promise.resolve({metadata: {}, items: []}),
            () => services.workflows.watchResources(namespace, `workflows.argoproj.io/workflow=${name}`),
            () => setError(null),
            () => setError(null),
            items => setResources([...items]),
            setError
        );
        lw.start();
        return () => lw.stop();
    }, [namespace, name]);

    const g = new Graph();
    const nodeGenres: {[genre: string]: boolean} = {Workflow: true};
    const nodeClassNames: {[className: string]: boolean} = {};

    g.nodes.set('main/' + namespace + '/' + name + '/argoproj.io/v1alpha1/Workflow', {genre: 'Workflow', label: name, icon: icons.Workflow});

    (resources || []).forEach(r => {
        const m = r.metadata;
        const node = m.clusterName + '/' + m.namespace + '/' + m.name + '/' + r.apiVersion + '/' + r.kind;
        const genre = r.kind;
        const classNames = r.status.phase;
        g.nodes.set(node, {genre, label: m.name, icon: icons[r.kind] || 'box', classNames});
        nodeGenres[genre] = true;
        nodeClassNames[classNames] = true;

        const group = m.clusterName + '/' + m.namespace;

        if (!g.nodeGroups.has(group)) {
            g.nodeGroups.set(group, new Set());
        }
        g.nodeGroups.get(group).add(node);

        (m.ownerReferences || []).forEach((o: {apiVersion: string; kind: string; name: string}) => {
            g.edges.set({v: m.clusterName + '/' + m.namespace + '/' + o.name + '/' + o.apiVersion + '/' + o.kind, w: node}, {});
        });
    });

    return (
        <>
            <ErrorNotice error={error} />
            <GraphPanel
                graph={g}
                nodeGenres={nodeGenres}
                nodeClassNames={nodeClassNames}
                storageScope='workflow-resources-graph'
                horizontal={true}
                classNames='workflow-resources-graph'
                nodeSize={48}
                selectedNode={selectedNode}
                onNodeSelect={setSelectedNode}
            />
            <SlidingPanel isShown={!!selectedNode} onClose={() => setSelectedNode(null)}>
                {selectedNode && (
                    <ObjectEditor
                        value={resources.find(
                            x => x.metadata.clusterName + '/' + x.metadata.namespace + '/' + x.metadata.name + '/' + x.apiVersion + '/' + x.kind === selectedNode
                        )}
                    />
                )}
            </SlidingPanel>
        </>
    );
};

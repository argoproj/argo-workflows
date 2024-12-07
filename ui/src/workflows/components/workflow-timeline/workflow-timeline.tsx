import classNames from 'classnames';
import React, {useEffect, useRef, useState} from 'react';
import {fromEvent, interval, Subscription} from 'rxjs';

import * as models from '../../../shared/models';
import {shortNodeName} from '../../utils';

import './workflow-timeline.scss';

const ROUND_START_DIFF_MS = 1000;
const NODE_NAME_WIDTH = 250;
const MIN_WIDTH = 800;
const COMPLETED_PHASES = [models.NODE_PHASE.ERROR, models.NODE_PHASE.SUCCEEDED, models.NODE_PHASE.SKIPPED, models.NODE_PHASE.OMITTED, models.NODE_PHASE.FAILED];

interface WorkflowTimelineProps {
    workflow: models.Workflow;
    selectedNodeId: string;
    nodeClicked?: (node: models.NodeStatus) => any;
}

function dateDiff(dateLikeA: string | number, dateLikeB: string | number): number {
    return new Date(dateLikeA).valueOf() - new Date(dateLikeB).valueOf();
}

function hhMMFormat(dateLike: string | number): string {
    const date = new Date(dateLike);
    // timeString format is '00:59:00 GMT-0500 (Eastern Standard Time)', hh:MM is '00:59'
    const parts = date.toTimeString().split(':');
    return parts[0] + ':' + parts[1];
}

export function WorkflowTimeline(props: WorkflowTimelineProps) {
    const [parentWidth, setParentWidth] = useState(0);
    const [now, setNow] = useState(new Date());

    const containerRef = useRef<HTMLDivElement>(null);
    const resizeSubscription = useRef<Subscription>(null);
    const refreshSubscription = useRef<Subscription>(null);

    useEffect(() => {
        resizeSubscription.current = fromEvent(window, 'resize').subscribe(updateWidth);
        updateWidth();

        return () => {
            resizeSubscription.current?.unsubscribe();
            refreshSubscription.current?.unsubscribe();
        };
    }, []);

    useEffect(() => {
        const isCompleted = props.workflow?.status && COMPLETED_PHASES.includes(props.workflow.status.phase);
        if (!refreshSubscription.current && !isCompleted) {
            refreshSubscription.current = interval(1000).subscribe(() => {
                setNow(new Date());
            });
        } else if (refreshSubscription.current && isCompleted) {
            refreshSubscription.current.unsubscribe();
            refreshSubscription.current = null;
        }
    }, [props.workflow]);

    function updateWidth() {
        if (containerRef.current) {
            setParentWidth((containerRef.current.offsetParent || window.document.body).clientWidth - NODE_NAME_WIDTH);
        }
    }

    if (!props.workflow.status.nodes) {
        return <p>No nodes</p>;
    }

    const nodes = Object.keys(props.workflow.status.nodes)
        .map(id => {
            const node = props.workflow.status.nodes[id];
            node.finishedAt = node.finishedAt || now.toISOString();
            node.startedAt = node.startedAt || now.toISOString();
            return node;
        })
        .filter(node => node.startedAt && node.type === 'Pod')
        .sort((first, second) => {
            const diff = dateDiff(first.startedAt, second.startedAt);
            if (diff <= 2) {
                return dateDiff(first.finishedAt, second.finishedAt);
            }
            return diff;
        });

    if (nodes.length === 0) {
        return <div />;
    }

    const timelineStart = new Date(nodes[0].startedAt).valueOf();
    const timelineEnd = nodes.map(node => new Date(node.finishedAt).valueOf()).reduce((first, second) => Math.max(first, second), new Date(timelineStart).valueOf());

    function timeToLeft(time: number) {
        return ((time - timelineStart) / (timelineEnd - timelineStart)) * Math.max(parentWidth, MIN_WIDTH) + NODE_NAME_WIDTH;
    }

    const groups = nodes.map(node => ({
        startedAt: new Date(node.startedAt).valueOf(),
        finishedAt: new Date(node.finishedAt).valueOf(),
        nodes: [
            Object.assign({}, node, {
                left: timeToLeft(new Date(node.startedAt).valueOf()),
                width: timeToLeft(new Date(node.finishedAt).valueOf()) - timeToLeft(new Date(node.startedAt).valueOf())
            })
        ]
    }));

    for (let i = groups.length - 1; i >= 1; i--) {
        const cur = groups[i];
        const next = groups[i - 1];
        if (dateDiff(cur.startedAt, next.finishedAt) < 0 && dateDiff(next.startedAt, cur.startedAt) < ROUND_START_DIFF_MS) {
            next.nodes = next.nodes.concat(cur.nodes);
            next.finishedAt = nodes.map(node => new Date(node.finishedAt).valueOf()).reduce((first, second) => Math.max(first, second), next.finishedAt.valueOf());
            groups.splice(i, 1);
        }
    }

    return (
        <div className='workflow-timeline' ref={containerRef} style={{width: Math.max(parentWidth, MIN_WIDTH) + NODE_NAME_WIDTH}}>
            <div style={{left: NODE_NAME_WIDTH}} className='workflow-timeline__start-line' />
            <div className='workflow-timeline__row workflow-timeline__row--header' />
            {groups.map(group => [
                <div style={{left: timeToLeft(group.startedAt)}} key={`group-${group.startedAt}`} className={classNames('workflow-timeline__start-line')}>
                    <span className='workflow-timeline__start-line__time'>{hhMMFormat(group.startedAt)}</span>
                </div>,
                ...group.nodes.map(node => (
                    <div
                        key={node.id}
                        className={classNames('workflow-timeline__row', {'workflow-timeline__row--selected': node.id === props.selectedNodeId})}
                        onClick={() => props.nodeClicked && props.nodeClicked(node)}>
                        <div className='workflow-timeline__node-name'>
                            <span title={shortNodeName(node)}>{shortNodeName(node)}</span>
                        </div>
                        <div style={{left: node.left, width: node.width}} className={`workflow-timeline__node workflow-timeline__node--${node.phase.toLocaleLowerCase()}`} />
                    </div>
                ))
            ])}
        </div>
    );
}

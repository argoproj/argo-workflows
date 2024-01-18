import classNames from 'classnames';
import moment from 'moment';
import React, {useEffect, useRef, useState} from 'react';
import {fromEvent, interval, Subscription} from 'rxjs';

import * as models from '../../../../models';
import {Utils} from '../../../shared/utils';

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

export function WorkflowTimeline(props: WorkflowTimelineProps) {
    const [parentWidth, setParentWidth] = useState(0);
    const [now, setNow] = useState(moment());

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
                setNow(moment());
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
            node.finishedAt = node.finishedAt || now.format();
            node.startedAt = node.startedAt || now.format();
            return node;
        })
        .filter(node => node.startedAt && node.type === 'Pod')
        .sort((first, second) => {
            const diff = moment(first.startedAt).diff(second.startedAt);
            if (diff <= 2) {
                return moment(first.finishedAt).diff(second.finishedAt);
            }
            return diff;
        });

    if (nodes.length === 0) {
        return <div />;
    }

    const timelineStart = moment(nodes[0].startedAt).valueOf();
    const timelineEnd = nodes.map(node => moment(node.finishedAt).valueOf()).reduce((first, second) => Math.max(first, second), moment(timelineStart).valueOf());

    function timeToLeft(time: number) {
        return ((time - timelineStart) / (timelineEnd - timelineStart)) * Math.max(parentWidth, MIN_WIDTH) + NODE_NAME_WIDTH;
    }

    const groups = nodes.map(node => ({
        startedAt: moment(node.startedAt).valueOf(),
        finishedAt: moment(node.finishedAt).valueOf(),
        nodes: [
            Object.assign({}, node, {
                left: timeToLeft(moment(node.startedAt).valueOf()),
                width: timeToLeft(moment(node.finishedAt).valueOf()) - timeToLeft(moment(node.startedAt).valueOf())
            })
        ]
    }));

    for (let i = groups.length - 1; i >= 1; i--) {
        const cur = groups[i];
        const next = groups[i - 1];
        if (moment(cur.startedAt).diff(next.finishedAt, 'milliseconds') < 0 && moment(next.startedAt).diff(cur.startedAt, 'milliseconds') < ROUND_START_DIFF_MS) {
            next.nodes = next.nodes.concat(cur.nodes);
            next.finishedAt = nodes.map(node => moment(node.finishedAt).valueOf()).reduce((first, second) => Math.max(first, second), next.finishedAt.valueOf());
            groups.splice(i, 1);
        }
    }

    return (
        <div className='workflow-timeline' ref={containerRef} style={{width: Math.max(parentWidth, MIN_WIDTH) + NODE_NAME_WIDTH}}>
            <div style={{left: NODE_NAME_WIDTH}} className='workflow-timeline__start-line' />
            <div className='workflow-timeline__row workflow-timeline__row--header' />
            {groups.map(group => [
                <div style={{left: timeToLeft(group.startedAt)}} key={`group-${group.startedAt}`} className={classNames('workflow-timeline__start-line')}>
                    <span className='workflow-timeline__start-line__time'>{moment(group.startedAt).format('hh:mm')}</span>
                </div>,
                ...group.nodes.map(node => (
                    <div
                        key={node.id}
                        className={classNames('workflow-timeline__row', {'workflow-timeline__row--selected': node.id === props.selectedNodeId})}
                        onClick={() => props.nodeClicked && props.nodeClicked(node)}>
                        <div className='workflow-timeline__node-name'>
                            <span title={Utils.shortNodeName(node)}>{Utils.shortNodeName(node)}</span>
                        </div>
                        <div style={{left: node.left, width: node.width}} className={`workflow-timeline__node workflow-timeline__node--${node.phase.toLocaleLowerCase()}`} />
                    </div>
                ))
            ])}
        </div>
    );
}

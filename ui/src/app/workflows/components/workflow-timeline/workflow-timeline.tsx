import classNames from 'classnames';
import moment from 'moment';
import * as React from 'react';
import {fromEvent, interval, Subscription} from 'rxjs';

import * as models from '../../../../models';
import {Utils} from '../../../shared/utils';

require('./workflow-timeline.scss');

const ROUND_START_DIFF_MS = 1000;
const NODE_NAME_WIDTH = 250;
const MIN_WIDTH = 800;

interface WorkflowTimelineProps {
    workflow: models.Workflow;
    selectedNodeId: string;
    nodeClicked?: (node: models.NodeStatus) => any;
}

interface WorkflowTimelineState {
    parentWidth: number;
    now: moment.Moment;
}

export class WorkflowTimeline extends React.Component<WorkflowTimelineProps, WorkflowTimelineState> {
    private container: HTMLElement;
    private resizeSubscription: Subscription;
    private refreshSubscription: Subscription;

    constructor(props: WorkflowTimelineProps) {
        super(props);
        this.state = {parentWidth: 0, now: moment()};
        this.ensureRunningWorkflowRefreshing(props.workflow);
    }

    public componentDidMount() {
        this.resizeSubscription = fromEvent(window, 'resize').subscribe(() => this.updateWidth());
        this.updateWidth();
    }

    public componentWillReceiveProps(nextProps: WorkflowTimelineProps) {
        this.ensureRunningWorkflowRefreshing(nextProps.workflow);
    }

    public componentWillUnmount() {
        if (this.resizeSubscription) {
            this.resizeSubscription.unsubscribe();
            this.resizeSubscription = null;
        }
        if (this.refreshSubscription) {
            this.refreshSubscription.unsubscribe();
            this.refreshSubscription = null;
        }
    }

    public render() {
        if (!this.props.workflow.status.nodes) {
            return <p>No nodes</p>;
        }
        const nodes = Object.keys(this.props.workflow.status.nodes)
            .map(id => {
                const node = this.props.workflow.status.nodes[id];
                node.finishedAt = node.finishedAt || this.state.now.format();
                node.startedAt = node.startedAt || this.state.now.format();
                return node;
            })
            .filter(node => node.startedAt && node.type === 'Pod')
            .sort((first, second) => {
                const diff = moment(first.startedAt).diff(second.startedAt);
                // If node started almost at the same time then sort by end time
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

        const timeToLeft = (time: number) => ((time - timelineStart) / (timelineEnd - timelineStart)) * Math.max(this.state.parentWidth, MIN_WIDTH) + NODE_NAME_WIDTH;
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
            <div className='workflow-timeline' ref={container => (this.container = container)} style={{width: Math.max(this.state.parentWidth, MIN_WIDTH) + NODE_NAME_WIDTH}}>
                <div style={{left: NODE_NAME_WIDTH}} className='workflow-timeline__start-line' />
                <div className='workflow-timeline__row workflow-timeline__row--header' />
                {groups.map(group => [
                    <div style={{left: timeToLeft(group.startedAt)}} key={`group-${group.startedAt}`} className={classNames('workflow-timeline__start-line')}>
                        <span className='workflow-timeline__start-line__time'>{moment(group.startedAt).format('hh:mm')}</span>
                    </div>,
                    ...group.nodes.map(node => (
                        <div
                            key={node.id}
                            className={classNames('workflow-timeline__row', {'workflow-timeline__row--selected': node.id === this.props.selectedNodeId})}
                            onClick={() => this.props.nodeClicked && this.props.nodeClicked(node)}>
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

    public updateWidth() {
        if (this.container) {
            this.setState({parentWidth: (this.container.offsetParent || window.document.body).clientWidth - NODE_NAME_WIDTH});
        }
    }

    private ensureRunningWorkflowRefreshing(workflow: models.Workflow) {
        const completedPhases = [models.NODE_PHASE.ERROR, models.NODE_PHASE.SUCCEEDED, models.NODE_PHASE.SKIPPED, models.NODE_PHASE.OMITTED, models.NODE_PHASE.FAILED];
        const isCompleted = workflow && workflow.status && completedPhases.indexOf(workflow.status.phase) > -1;
        if (!this.refreshSubscription && !isCompleted) {
            this.refreshSubscription = interval(1000).subscribe(() => {
                this.setState({now: moment()});
            });
        } else if (this.refreshSubscription && isCompleted) {
            this.refreshSubscription.unsubscribe();
            this.refreshSubscription = null;
        }
    }
}

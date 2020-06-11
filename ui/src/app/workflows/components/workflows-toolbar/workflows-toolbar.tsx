import * as React from 'react';
require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: boolean};
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, {}> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
    }

    public render() {
        return <div className='workflows-toolbar'> {this.getNumberSelected()} workflows selected</div>;
    }

    private getNumberSelected(): number {
        return Object.keys(this.props.selectedWorkflows).length;
    }
}

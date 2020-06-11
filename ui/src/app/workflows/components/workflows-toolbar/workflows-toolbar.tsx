import * as React from 'react';

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: boolean};
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, {}> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
    }

    public render() {
        return <div> {this.getNumberSelected()} selected</div>;
    }

    private getNumberSelected(): number {
        let count = 0;
        for (const wf of Object.keys(this.props.selectedWorkflows)) {
            if (this.props.selectedWorkflows[wf]) {
                count++;
            }
        }
        return count;
    }
}

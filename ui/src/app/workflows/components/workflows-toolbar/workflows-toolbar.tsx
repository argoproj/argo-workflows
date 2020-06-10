import * as React from 'react';
import * as models from '../../../../models';

interface WorkflowsToolbarProps {
    selectedWorkflows: models.Workflow[];
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, {}> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
    }

    public render() {
        const wfList = this.props.selectedWorkflows;
        return <div> {wfList.length || 'No'} workflows selected</div>;
    }
}

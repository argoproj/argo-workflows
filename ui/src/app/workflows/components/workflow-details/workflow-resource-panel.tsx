import * as React from 'react';
import {Workflow} from '../../../../models';
import {ResourceViewer} from '../../../shared/components/resource-editor/resource-viewer';

export const WorkflowResourcePanel = (props: {workflow: Workflow}) => (
    <div className='white-box'>
        <div className='white-box__details'>
            <ResourceViewer value={props.workflow} />
        </div>
    </div>
);

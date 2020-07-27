import * as React from 'react';
import {Workflow} from '../../../../models';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';

export const WorkflowResourcePanel = (props: {workflow: Workflow}) => (
    <div className='white-box' key='workflow-resource'>
        <div className='white-box__details'>
            <ResourceEditor readonly={true} value={props.workflow} kind='Workflow' />
        </div>
    </div>
);

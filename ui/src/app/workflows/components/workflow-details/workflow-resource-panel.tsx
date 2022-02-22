import * as React from 'react';
import {Workflow} from '../../../../models';
import {ObjectEditor} from '../../../shared/components/object-editor/object-editor';

export const WorkflowResourcePanel = (props: {workflow: Workflow}) => (
    <div className='white-box'>
        <ObjectEditor value={props.workflow} type='io.argoproj.workflow.v1alpha1.Workflow' />
    </div>
);

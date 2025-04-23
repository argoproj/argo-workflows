import * as React from 'react';

import {SerializingObjectEditor} from '../../../shared/components/object-editor';
import {Workflow} from '../../../shared/models';

export const WorkflowResourcePanel = (props: {workflow: Workflow}) => (
    <div className='white-box'>
        <SerializingObjectEditor value={props.workflow} type='io.argoproj.workflow.v1alpha1.Workflow' />
    </div>
);

import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {exampleTemplate, randomSillyName} from '../../examples';
import {WorkflowSpecPanel} from '../workflow-spec-panel/workflow-spec-panel';

export const WorkflowSpecEditor = (props: {value: WorkflowSpec; onChange: (value: WorkflowSpec) => void}) => {
    return (
        <div key='workflow-spec-editor' className='white-box'>
            <h5>Specification</h5>
            <div>
                <button
                    className='argo-button argo-button--base-o'
                    onClick={() => {
                        props.value.templates.push(exampleTemplate(randomSillyName()));
                        props.onChange(props.value);
                    }}>
                    <i className='fa fa-box' /> Add container template
                </button>
            </div>
            <WorkflowSpecPanel spec={props.value} />
        </div>
    );
};

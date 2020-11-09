import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {exampleTemplate, randomSillyName} from '../../examples';
import {WorkflowSpecPanel} from '../workflow-spec-panel/workflow-spec-panel';

export const WorkflowSpecEditor = (props: {value: WorkflowSpec; onChange: (value: WorkflowSpec) => void}) => {
    const [spec, setSpec] = React.useState(props.value);
    React.useEffect(() => {
        props.onChange(spec);
    }, [spec]);
    return (
        <div key='workflow-spec-editor' className='white-box'>
            <h4>Specification</h4>
            <div>
                <button
                    className='argo-button argo-button--base-o'
                    onClick={() => {
                        setSpec(s => {
                            s.templates.push(exampleTemplate(randomSillyName()));
                            return s;
                        });
                    }}>
                    <i className='fa fa-box' /> Add container template
                </button>
            </div>
            <WorkflowSpecPanel spec={spec} />
        </div>
    );
};

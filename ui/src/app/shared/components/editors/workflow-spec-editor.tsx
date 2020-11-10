import {SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {exampleTemplate, randomSillyName} from '../../examples';
import {ResourceEditor} from '../resource-editor/resource-editor';
import {ID} from '../workflow-spec-panel/id';
import {WorkflowSpecPanel} from '../workflow-spec-panel/workflow-spec-panel';

export const WorkflowSpecEditor = (props: {value: WorkflowSpec; onChange: (value: WorkflowSpec) => void}) => {
    const [selectedId, setSelectedId] = React.useState<string>();
    const kind = (id: string) => {
        const {type} = ID.split(id);
        const kinds: {[key: string]: string} = {
            Artifacts: 'Artifacts',
            Parameters: 'Parameters',
            Step: 'WorkflowStep',
            Template: 'Template',
            Task: 'DagTask',
            Workflow: 'WorkflowSpec'
        };
        return kinds[type];
    };
    const object = (id: string) => {
        const {type, name} = ID.split(id);
        const template = () => props.value.templates.find(t => t.name === name);
        switch (type) {
            case 'Artifacts':
                return props.value.arguments.artifacts;
            case 'OnExit':
                return props.value.onExit;
            case 'Parameters':
                return props.value.arguments.parameters;
            case 'Step':
                return template();
            case 'StepGroup':
                return template();
            case 'Task':
                return template();
            case 'Template':
                return template();
            case 'TemplateRef':
                return template();
            case 'WorkflowTemplateRef':
                return props.value.workflowTemplateRef;
            case 'Workflow':
                return props.value;
            default:
                return null;
        }
    };
    const setObject = (id: string, value: any) => {
        const {type, name} = ID.split(id);
        const i = props.value.templates.findIndex(t => t.name === name);
        switch (type) {
            case 'Artifacts':
                props.value.arguments.artifacts = value;
                break;
            case 'OnExit':
                props.value.onExit = value;
                break;
            case 'Parameters':
                props.value.arguments.parameters = value;
                break;
            case 'Step':
                props.value.templates[i] = value;
                break;
            case 'StepGroup':
                props.value.templates[i] = value;
                break;
            case 'Task':
                props.value.templates[i] = value;
                break;
            case 'Template':
                props.value.templates[i] = value;
                break;
            case 'TemplateRef':
                props.value.templates[i] = value;
                break;
            case 'WorkflowTemplateRef':
                props.value.workflowTemplateRef = value;
                break;
            case 'Workflow':
                props.value = value;
                break;
        }
    };
    return (
        <div key='workflow-spec-editor' className='white-box'>
            <h5>Specification</h5>
            <div>
                <button
                    className='argo-button argo-button--base-o'
                    onClick={() => {
                        const templateName = randomSillyName();
                        props.value.templates.push(exampleTemplate(templateName));
                        props.onChange(props.value);
                        setSelectedId(ID.join('Template', templateName));
                    }}>
                    <i className='fa fa-box' /> Add container template
                </button>
            </div>
            <WorkflowSpecPanel spec={props.value} selectedId={selectedId} onSelect={id => setSelectedId(id)} />
            <SlidingPanel isShown={selectedId !== undefined} onClose={() => setSelectedId(undefined)}>
                {selectedId && object(selectedId) && (
                    <>
                        <h4>{selectedId}</h4>
                        <div>
                            <ResourceEditor
                                kind={kind(selectedId)}
                                value={object(selectedId)}
                                editing={true}
                                onSubmit={value => Promise.resolve(setObject(selectedId, value)).then(() => setSelectedId(undefined))}
                            />
                        </div>
                    </>
                )}
            </SlidingPanel>
        </div>
    );
};

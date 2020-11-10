import {SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {WorkflowSpec} from '../../../../models';
import {exampleTemplate, randomSillyName} from '../../examples';
import {Button} from '../button';
import {ResourceEditor} from '../resource-editor/resource-editor';
import {icons} from '../workflow-spec-panel/icons';
import {idForTemplate, onExitId, stepGroupOf, stepOf, taskOf, templateOf, typeOf} from '../workflow-spec-panel/id';
import {WorkflowSpecPanel} from '../workflow-spec-panel/workflow-spec-panel';

export const WorkflowSpecEditor = (props: {value: WorkflowSpec; onChange: (value: WorkflowSpec) => void}) => {
    const [selectedId, setSelectedId] = React.useState<string>();
    const kind = (id: string) => {
        const type = typeOf(id);
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
        const type = typeOf(id);
        const template = (name: string) => props.value.templates.find(t => t.name === name);
        switch (type) {
            case 'Artifacts':
                return props.value.arguments.artifacts;
            case 'OnExit':
                return props.value.onExit;
            case 'Parameters':
                return props.value.arguments.parameters;
            case 'Step': {
                const {templateName, i, j} = stepOf(id);
                return template(templateName).steps[i][j];
            }
            case 'StepGroup': {
                const {templateName, i} = stepGroupOf(id);
                return template(templateName).steps[i];
            }
            case 'Task': {
                const {templateName, taskName} = taskOf(id);
                return template(templateName).dag.tasks.find(task => task.name === taskName);
            }
            case 'Template':
                return template(templateOf(id).templateName);
            case 'WorkflowTemplateRef':
                return props.value.workflowTemplateRef;
        }
    };
    const setObject = (id: string, value: any) => {
        const type = typeOf(id);
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
                {
                    const {templateName, i, j} = stepOf(id);
                    props.value.templates.find(t => t.name === templateName).steps[i][j] = value;
                }
                break;
            case 'StepGroup':
                {
                    const {templateName, i} = stepGroupOf(id);
                    props.value.templates.find(t => t.name === templateName).steps[i] = value;
                }
                break;
            case 'Task':
                {
                    const {templateName, taskName} = taskOf(id);
                    const tasks = props.value.templates.find(t => t.name === templateName).dag.tasks;
                    const i = tasks.findIndex(t => t.name === taskName);
                    tasks[i] = value;
                }
                break;
            case 'Template':
                {
                    const {templateName} = templateOf(id);
                    const i = props.value.templates.findIndex(t => t.name === templateName);
                    props.value.templates[i] = value;
                }
                break;
            case 'WorkflowTemplateRef':
                props.value.workflowTemplateRef = value;
                break;
        }
    };
    const deleteObject = (id: string) => {
        const type = typeOf(id);
        switch (type) {
            case 'Artifacts':
                delete props.value.arguments.artifacts;
                break;
            case 'OnExit':
                delete props.value.onExit;
                break;
            case 'Parameters':
                delete props.value.arguments.parameters;
                break;
            case 'Step':
                {
                    const {templateName, i, j} = stepOf(id);
                    delete props.value.templates.find(t => t.name === templateName).steps[i][j];
                }
                break;
            case 'StepGroup':
                {
                    const {templateName, i} = stepGroupOf(id);
                    delete props.value.templates.find(t => t.name === templateName).steps[i];
                }
                break;
            case 'Task':
                {
                    const {templateName, taskName} = taskOf(id);
                    const tasks = props.value.templates.find(t => t.name === templateName).dag.tasks;
                    const i = tasks.findIndex(t => t.name === taskName);
                    delete tasks[i];
                }
                break;
            case 'Template':
                {
                    const {templateName} = templateOf(id);
                    const i = props.value.templates.findIndex(t => t.name === templateName);
                    delete props.value.templates[i];
                }
                break;
            case 'WorkflowTemplateRef':
                delete props.value.workflowTemplateRef;
                break;
        }
    };
    const anyContainerOrScriptTemplate = () => props.value.templates.find(t => t.container || t.script);
    const bestTemplateName = () => (anyContainerOrScriptTemplate() || {name: 'TBD'}).name;
    return (
        <div key='workflow-spec-editor' className='white-box'>
            <h5>Specification</h5>
            <label>
                Add{' '}
                <Button
                    icon={icons.container}
                    onClick={() => {
                        const templateName = randomSillyName();
                        props.value.templates.push(exampleTemplate(templateName));
                        props.onChange(props.value);
                        setSelectedId(idForTemplate(templateName));
                    }}>
                    Container
                </Button>
                <Button
                    icon={icons.script}
                    onClick={() => {
                        const templateName = randomSillyName();
                        props.value.templates.push({
                            name: templateName,
                            inputs: {
                                parameters: [{name: 'message', value: '{{workflow.parameters.message}}'}]
                            },
                            script: {
                                image: 'docker/whalesay:latest',
                                command: ['sh'],
                                source: 'echo {{inputs.parameters.message}}'
                            }
                        });
                        props.onChange(props.value);
                        setSelectedId(idForTemplate(templateName));
                    }}>
                    Script
                </Button>
                <Button
                    icon={icons.dag}
                    onClick={() => {
                        const templateName = randomSillyName();
                        props.value.templates.push({
                            name: templateName,
                            dag: {
                                tasks: [
                                    {
                                        name: 'main',
                                        template: bestTemplateName()
                                    }
                                ]
                            }
                        });
                        props.onChange(props.value);
                        setSelectedId(idForTemplate(templateName));
                    }}>
                    DAG
                </Button>
                <Button
                    icon={icons.steps}
                    onClick={() => {
                        const templateName = randomSillyName();
                        props.value.templates.push({
                            name: templateName,
                            steps: [
                                [
                                    {
                                        name: 'main',
                                        template: bestTemplateName()
                                    }
                                ]
                            ]
                        });
                        props.onChange(props.value);
                        setSelectedId(idForTemplate(templateName));
                    }}>
                    Steps
                </Button>
                <Button
                    icon={icons.onExit}
                    onClick={() => {
                        props.value.onExit = bestTemplateName();
                        props.onChange(props.value);
                        setSelectedId(onExitId);
                    }}>
                    Exit handler
                </Button>
            </label>
            <WorkflowSpecPanel spec={props.value} selectedId={selectedId} onSelect={id => setSelectedId(id)} />
            <SlidingPanel isShown={selectedId !== undefined} onClose={() => setSelectedId(undefined)}>
                {selectedId && object(selectedId) ? (
                    <>
                        <h4>{selectedId}</h4>
                        <div>
                            <ResourceEditor
                                kind={kind(selectedId)}
                                value={object(selectedId)}
                                editing={true}
                                onSubmit={value => Promise.resolve(setObject(selectedId, value)).then(() => setSelectedId(undefined))}
                                onDelete={() => Promise.resolve(deleteObject(selectedId)).then(() => setSelectedId(undefined))}
                            />
                        </div>
                    </>
                ) : (
                    <>
                        <h4>Specification</h4>
                        <ResourceEditor
                            kind='WorkflowSpec'
                            value={props.value}
                            editing={true}
                            onSubmit={value => Promise.resolve(props.onChange(value)).then(() => setSelectedId(undefined))}
                        />
                    </>
                )}
            </SlidingPanel>
        </div>
    );
};

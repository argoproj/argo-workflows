import {SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Arguments, Parameter, WorkflowSpec} from '../../../../models';
import {exampleTemplate, randomSillyName} from '../../examples';
import {Button} from '../button';
import {ObjectEditor} from '../object-editor/object-editor';
import {icons} from '../workflow-spec-panel/icons';
import {idForTemplate, onExitId, stepGroupOf, stepOf, taskOf, templateOf, typeOf} from '../workflow-spec-panel/id';
import {WorkflowSpecPanel} from '../workflow-spec-panel/workflow-spec-panel';
import {KeyValueEditor} from './key-value-editor';

require('./workflow-spec-editor.scss');

const type = (id: string) => {
    const types: {[key: string]: string} = {
        Step: 'io.argoproj.workflow.v1alpha1.WorkflowStep',
        Template: 'io.argoproj.workflow.v1alpha1.Template',
        Task: 'io.argoproj.workflow.v1alpha1.DagTask',
        Workflow: 'io.argoproj.workflow.v1alpha1.WorkflowSpec'
    };
    return types[typeOf(id)];
};

export const WorkflowSpecEditor = <T extends WorkflowSpec>(props: {value: T; onChange: (value: T) => void; onError: (error: Error) => void}) => {
    const [selectedId, setSelectedId] = React.useState<string>();

    const templateByName = (name: string) => props.value.templates.filter(t => !!t).find(t => t.name === name);

    const object = (id: string) => {
        switch (typeOf(id)) {
            case 'Artifacts':
                return props.value.arguments.artifacts;
            case 'OnExit':
                return props.value.onExit;
            case 'Parameters':
                return props.value.arguments.parameters;
            case 'Step': {
                const {templateName, i, j} = stepOf(id);
                return templateByName(templateName).steps[i][j];
            }
            case 'StepGroup': {
                const {templateName, i} = stepGroupOf(id);
                return templateByName(templateName).steps[i];
            }
            case 'Task': {
                const {templateName, taskName} = taskOf(id);
                return templateByName(templateName).dag.tasks.find(task => task.name === taskName);
            }
            case 'Template':
                return templateByName(templateOf(id).templateName);
            case 'WorkflowTemplateRef':
                return props.value.workflowTemplateRef;
        }
    };
    const setObject = (id: string, value: any) => {
        switch (typeOf(id)) {
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
        switch (typeOf(id)) {
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
                    props.value.templates.find(t => t.name === templateName).steps[i].splice(j, 1);
                }
                break;
            case 'StepGroup':
                {
                    const {templateName, i} = stepGroupOf(id);
                    props.value.templates.find(t => t.name === templateName).steps.splice(i, 1);
                }
                break;
            case 'Task':
                {
                    const {templateName, taskName} = taskOf(id);
                    const tasks = props.value.templates.find(t => t.name === templateName).dag.tasks;
                    const i = tasks.findIndex(t => t.name === taskName);
                    tasks.splice(i, 1);
                }
                break;
            case 'Template':
                {
                    const {templateName} = templateOf(id);
                    const i = props.value.templates.findIndex(t => t.name === templateName);
                    props.value.templates.splice(i, 1);
                }
                break;
            case 'WorkflowTemplateRef':
                delete props.value.workflowTemplateRef;
                break;
        }
    };
    const bestTemplateName = () => (props.value.templates.find(t => t.container || t.script) || {name: 'TBD'}).name;
    const parameterKeyValues =
        props.value &&
        props.value.arguments &&
        props.value.arguments.parameters &&
        props.value.arguments.parameters
            .map(param => [param.name, param.value])
            .reduce((obj, [key, val]) => {
                obj[key] = val;
                return obj;
            }, {} as {[key: string]: string});
    const selectedObject = object(selectedId);
    const selectedType = type(selectedId);
    return (
        <>
            <div className='white-box'>
                <h5>Parameters</h5>
                <KeyValueEditor
                    keyValues={parameterKeyValues}
                    onChange={parameters => {
                        if (!props.value.arguments) {
                            props.value.arguments = {parameters: []} as Arguments;
                        }
                        props.value.arguments.parameters = Object.entries(parameters).map(
                            ([k, v]) =>
                                ({
                                    name: k,
                                    value: v
                                } as Parameter)
                        );
                        props.onChange(props.value);
                    }}
                />
            </div>
            <div className='white-box'>
                <div className='row'>
                    <div className='columns xlarge-11'>
                        <WorkflowSpecPanel spec={props.value} selectedId={selectedId} onSelect={setSelectedId} />
                    </div>
                    <div className='columns xlarge-1'>
                        <div className='object-palette'>
                            <a
                                title='Container'
                                onClick={() => {
                                    const templateName = randomSillyName();
                                    props.value.templates.push(exampleTemplate(templateName));
                                    props.onChange(props.value);
                                    setSelectedId(idForTemplate(templateName));
                                }}>
                                <i className={'fa fa-' + icons.container} />{' '}
                            </a>
                            <a
                                title='Script'
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
                                <i className={'fa fa-' + icons.script} />
                            </a>
                            <a
                                title='DAG'
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
                                <i className={'fa fa-' + icons.dag} />
                            </a>
                            <a
                                title='Steps'
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
                                <i className={'fa fa-' + icons.steps} />
                            </a>
                            <a
                                title='Exit handler'
                                onClick={() => {
                                    props.value.onExit = bestTemplateName();
                                    props.onChange(props.value);
                                    setSelectedId(onExitId);
                                }}>
                                <i className={'fa fa-' + icons.onExit} />
                            </a>
                        </div>
                    </div>
                </div>
                <SlidingPanel isShown={!!selectedId} onClose={() => setSelectedId(null)} isNarrow={true}>
                    {selectedObject ? (
                        <>
                            <h4>{selectedId}</h4>
                            <ObjectEditor
                                type={selectedType}
                                value={selectedObject}
                                onChange={value => setObject(selectedId, value)}
                                buttons={
                                    <>
                                        <Button
                                            icon='times'
                                            onClick={() => {
                                                setSelectedId(undefined);
                                                deleteObject(selectedId);
                                            }}>
                                            Remove
                                        </Button>
                                        <Button icon='check' onClick={() => setSelectedId(undefined)}>
                                            OK
                                        </Button>
                                    </>
                                }
                            />
                        </>
                    ) : (
                        <>
                            <h4>Specification</h4>
                            <ObjectEditor type={type('WorkflowSpec')} value={props.value} onChange={props.onChange} />
                        </>
                    )}
                </SlidingPanel>
            </div>
        </>
    );
};

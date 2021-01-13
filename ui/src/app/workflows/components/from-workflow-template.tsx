import {Select} from 'argo-ui/src/components/select/select';
import {useEffect, useState} from 'react';
import * as React from 'react';
import {Workflow, WorkflowTemplate} from '../../../models';
import {services} from '../../shared/services';
import {SubmitWorkflowPanel} from './submit-workflow-panel';

const workflowFromWorkflowTemplateName = (templateName: string, namespace: string): Workflow => {
    return {
        metadata: {
            generateName: templateName + '-',
            namespace,
            labels: {
                'workflows.argoproj.io/workflow-template': templateName
            }
        },
        spec: {
            workflowTemplateRef: {
                name: templateName
            }
        }
    };
};

export const FromWorkflowTemplate = ({
    namespace,
    onError,
    onTemplateSelect
}: {
    namespace: string;
    onError: (error: Error) => void;
    onTemplateSelect: (workflowWithTemplate: Workflow) => void;
}) => {
    const [workflowTemplates, setWorkflowTemplates] = useState<WorkflowTemplate[]>();
    const [workflowTemplate, setWorkflowTemplate] = useState<WorkflowTemplate>();

    useEffect(() => {
        services.workflowTemplate
            .list(namespace)
            .then(setWorkflowTemplates)
            .catch(onError);
    }, []);

    return (
        <>
            <div className={'white-box'}>
                <label>Submit From Workflow Template</label>
                <Select
                    options={workflowTemplates && workflowTemplates.length > 0 ? workflowTemplates.map(tmpl => tmpl.metadata.name) : []}
                    value={workflowTemplate ? workflowTemplate.metadata.name : ''}
                    onChange={templateName => {
                        setWorkflowTemplate(workflowTemplates.find(template => template.metadata.name === templateName.title));
                        onTemplateSelect(workflowFromWorkflowTemplateName(templateName.title, namespace));
                    }}
                />
            </div>
            <div>
                {workflowTemplate && (
                    <SubmitWorkflowPanel
                        kind='WorkflowTemplate'
                        namespace={workflowTemplate.metadata.namespace}
                        name={workflowTemplate.metadata.name}
                        entrypoint={workflowTemplate.spec.entrypoint}
                        entrypoints={(workflowTemplate.spec.templates || []).map(t => t.name)}
                        parameters={workflowTemplate.spec.arguments.parameters || []}
                    />
                )}
            </div>
        </>
    );
};

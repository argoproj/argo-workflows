import {Select} from 'argo-ui/src/components/select/select';
import {useEffect, useState} from 'react';
import * as React from 'react';
import {Workflow, WorkflowTemplate} from '../../../models';
import {services} from '../../shared/services';

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

    useEffect(() => {
        services.workflowTemplate
            .list(namespace)
            .then(setWorkflowTemplates)
            .catch(onError);
    }, []);

    return (
        <div className={'white-box'}>
            <label>Submit From Workflow Template</label>
            <Select
                options={workflowTemplates && workflowTemplates.length > 0 ? workflowTemplates.map(tmpl => tmpl.metadata.name) : []}
                onChange={templateName => onTemplateSelect(workflowFromWorkflowTemplateName(templateName.title, namespace))}
            />
        </div>
    );
};

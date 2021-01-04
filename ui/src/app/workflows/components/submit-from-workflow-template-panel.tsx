import * as React from 'react';
import {useEffect, useState} from 'react';
import {WorkflowTemplate} from '../../../models';
import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {services} from '../../shared/services';
import {SubmitWorkflowPanel} from './submit-workflow-panel';

export const SubmitFromWorkflowTemplatePanel = ({namespace}: {namespace: string}) => {
    const [error, setError] = useState<Error>();
    const [workflowTemplates, setWorkflowTemplates] = useState<WorkflowTemplate[]>();
    const [workflowTemplate, setWorkflowTemplate] = useState<WorkflowTemplate>();

    useEffect(() => {
        services.workflowTemplate
            .list(namespace)
            .then(setWorkflowTemplates)
            .catch(setError);
    }, []);

    return (
        <>
            <ErrorNotice error={error} />
            {!workflowTemplate ? (
                <>
                    <h3>Select workflow template...</h3>
                    {workflowTemplates && workflowTemplates.length > 0 ? (
                        <ul>
                            {(workflowTemplates || []).map(x => (
                                <li>
                                    <a onClick={() => setWorkflowTemplate(x)}>{x.metadata.name}</a>
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <p>No templates found.</p>
                    )}
                </>
            ) : (
                <>
                    <div>
                        <Button icon='arrow-left' outline={true} onClick={() => setWorkflowTemplate(null)}>
                            Back
                        </Button>
                    </div>
                    <SubmitWorkflowPanel
                        kind='WorkflowTemplate'
                        namespace={namespace}
                        name={workflowTemplate.metadata.name}
                        entrypoint={workflowTemplate.spec.entrypoint}
                        entrypoints={(workflowTemplate.spec.templates || []).map(t => t.name) || []}
                        parameters={(workflowTemplate.spec.arguments || {}).parameters}
                    />
                </>
            )}
        </>
    );
};

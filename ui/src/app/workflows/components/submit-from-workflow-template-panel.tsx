import * as React from 'react';
import {useState} from 'react';
import {WorkflowTemplate} from '../../../models';
import {Button} from '../../shared/components/button';
import {DataLoaderDropdown} from '../../shared/components/data-loader-dropdown';
import {ErrorNotice} from '../../shared/components/error-notice';
import {services} from '../../shared/services';
import {SubmitWorkflowPanel} from './submit-workflow-panel';

export const SubmitFromWorkflowTemplatePanel = ({namespace}: {namespace: string}) => {
    const [error, setError] = useState<Error>();
    const [workflowTemplate, setWorkflowTemplate] = useState<WorkflowTemplate>();

    return (
        <>
            <ErrorNotice error={error} />
            {!workflowTemplate ? (
                <DataLoaderDropdown
                    load={() => services.workflowTemplate.list(namespace).then(list => list.map(x => x.metadata.name))}
                    onChange={name => {
                        services.workflowTemplate
                            .get(name, namespace)
                            .then(setWorkflowTemplate)
                            .catch(setError);
                    }}
                    placeholder='Select workflow template...'
                />
            ) : (
                <>
                    <div>
                        <Button icon='arrow-left' onClick={() => setWorkflowTemplate(null)}>
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

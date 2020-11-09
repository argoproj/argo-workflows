import * as React from 'react';
import {WorkflowTemplate} from '../../../../models';
import {services} from '../../services';
import {Button} from '../button';
import {Status, StatusNotice} from '../status-notice';
import {MetadataEditor} from './metadata-editor';
import {WorkflowSpecEditor} from './workflow-spec-editor';

export const WorkflowTemplateEditor = (props: {value: WorkflowTemplate}) => {
    const [template, setTemplate] = React.useState<WorkflowTemplate>(props.value);
    const [status, setStatus] = React.useState<Status>('Pending');
    const save = () => {
        services.workflowTemplate
            .update(template, template.metadata.name, template.metadata.namespace)
            .then(t => setTemplate(t))
            .then(() => setStatus('Succeeded'))
            .catch(error => setStatus(error));
    };
    return (
        <div>
            <div>
                <Button icon='save' onClick={() => save()}>
                    Save
                </Button>
            </div>
            <StatusNotice status={status} />
            <MetadataEditor
                value={template.metadata}
                onChange={value => {
                    setTemplate(s => {
                        s.metadata = value;
                        return s;
                    });
                }}
            />
            <WorkflowSpecEditor
                value={template.spec}
                onChange={value => {
                    setTemplate(s => {
                        s.spec = value;
                        return s;
                    });
                }}
            />
        </div>
    );
};

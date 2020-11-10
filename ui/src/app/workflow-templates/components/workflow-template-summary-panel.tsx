import * as React from 'react';

import {WorkflowTemplate} from '../../../models';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {WorkflowSpecEditor} from '../../shared/components/editors/workflow-spec-editor';

export const WorkflowTemplateSummaryPanel = (props: {template: WorkflowTemplate; onChange: (template: WorkflowTemplate) => void}) => {
    return (
        <div>
            <MetadataEditor value={props.template.metadata} onChange={metadata => props.onChange({...props.template, metadata})} />
            <WorkflowSpecEditor value={props.template.spec} onChange={spec => props.onChange({...props.template, spec})} />
        </div>
    );
};

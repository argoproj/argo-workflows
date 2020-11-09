import * as React from 'react';

import {WorkflowTemplate} from '../../../models';
import {WorkflowTemplateEditor} from '../../shared/components/editors/workflow-template-editor';

interface Props {
    template: WorkflowTemplate;
    onChange: (template: WorkflowTemplate) => void;
}

export const WorkflowTemplateSummaryPanel = (props: Props) => {
    return <WorkflowTemplateEditor value={props.template} />;
};

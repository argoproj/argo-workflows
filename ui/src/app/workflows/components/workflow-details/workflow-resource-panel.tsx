import {Tabs} from 'argo-ui';
import * as React from 'react';
import {Workflow} from '../../../../models';
import {ObjectEditor} from '../../../shared/components/object-editor/object-editor';
import {WorkflowSpecPanel} from '../../../shared/components/workflow-spec-panel/workflow-spec-panel';

export const WorkflowResourcePanel = (props: {workflow: Workflow}) => (
    <Tabs
        navTransparent={true}
        tabs={[
            {
                key: 'spec',
                title: 'Spec',
                content: <WorkflowSpecPanel spec={props.workflow.spec} />
            },
            {
                key: 'manifest',
                title: 'Manifest',
                content: <ObjectEditor value={props.workflow} type='io.argoproj.workflow.v1alpha1.Workflow' />
            }
        ]}
    />
);

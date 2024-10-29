import * as React from 'react';

import {ClusterWorkflowTemplateLink} from '../../cluster-workflow-templates/cluster-workflow-template-link';
import {CronWorkflowLink} from '../../cron-workflows/cron-workflow-link';
import {labels} from '../../shared/models';
import {WorkflowTemplateLink} from '../../workflow-templates/workflow-template-link';

export function WorkflowFrom(props: {namespace: string; labels: {[name: string]: string}}) {
    const workflowTemplate = props.labels[labels.workflowTemplate];
    const clusterWorkflowTemplate = props.labels[labels.clusterWorkflowTemplate];
    const cronWorkflow = props.labels[labels.cronWorkflow];
    return (
        <>
            {workflowTemplate && <WorkflowTemplateLink namespace={props.namespace} name={workflowTemplate} />}
            {clusterWorkflowTemplate && <ClusterWorkflowTemplateLink name={clusterWorkflowTemplate} />}
            {cronWorkflow && <CronWorkflowLink namespace={props.namespace} name={cronWorkflow} />}
            {!workflowTemplate && !clusterWorkflowTemplate && !cronWorkflow && '-'}
        </>
    );
}

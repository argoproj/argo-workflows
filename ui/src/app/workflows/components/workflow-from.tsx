import * as React from 'react';
import {labels} from '../../../models';
import {ClusterWorkflowTemplateLink} from '../../cluster-workflow-templates/components/cluster-workflow-template-link';
import {CronWorkflowLink} from '../../cron-workflows/components/cron-workflow-link';
import {WorkflowTemplateLink} from '../../workflow-templates/components/workflow-template-link';

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

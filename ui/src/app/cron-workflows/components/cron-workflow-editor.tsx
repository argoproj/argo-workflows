import * as React from 'react';

import {Tabs} from 'argo-ui';
import {CronWorkflow} from '../../../models';
import {LabelsAndAnnotationsEditor} from '../../shared/components/editors/labels-and-annotations-editor';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {WorkflowParametersEditor} from '../../shared/components/editors/workflow-parameters-editor';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';
import {CronWorkflowSpecEditor} from './cron-workflow-spec-editior';
import {CronWorkflowStatusViewer} from './cron-workflow-status-viewer';

export function CronWorkflowEditor({
    selectedTabKey,
    onTabSelected,
    onError,
    onChange,
    cronWorkflow
}: {
    cronWorkflow: CronWorkflow;
    onChange: (cronWorkflow: CronWorkflow) => void;
    onError: (error: Error) => void;
    onTabSelected?: (tab: string) => void;
    selectedTabKey?: string;
}) {
    return (
        <Tabs
            key='tabs'
            navTransparent={true}
            selectedTabKey={selectedTabKey}
            onTabSelected={onTabSelected}
            tabs={[
                ...(cronWorkflow.status
                    ? [
                          {
                              key: 'status',
                              title: 'Status',
                              content: <CronWorkflowStatusViewer spec={cronWorkflow.spec} status={cronWorkflow.status} />
                          }
                      ]
                    : []),
                {
                    key: 'manifest',
                    title: 'Manifest',
                    content: <ObjectEditor type='io.argoproj.workflow.v1alpha1.CronWorkflow' value={cronWorkflow} onChange={x => onChange({...x})} />
                },
                {
                    key: 'cron',
                    title: 'Cron',
                    content: <CronWorkflowSpecEditor spec={cronWorkflow.spec} onChange={spec => onChange({...cronWorkflow, spec})} />
                },
                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={cronWorkflow.metadata} onChange={metadata => onChange({...cronWorkflow, metadata})} />
                },
                {
                    key: 'workflow',
                    title: 'Workflow',
                    content: (
                        <WorkflowParametersEditor
                            value={cronWorkflow.spec.workflowSpec}
                            onChange={workflowSpec => onChange({...cronWorkflow, spec: {...cronWorkflow.spec, workflowSpec}})}
                            onError={onError}
                        />
                    )
                },
                {
                    key: 'workflow-metadata',
                    title: 'Workflow MetaData',
                    content: (
                        <LabelsAndAnnotationsEditor
                            value={cronWorkflow.spec.workflowMetadata}
                            onChange={workflowMetadata =>
                                onChange({
                                    ...cronWorkflow,
                                    spec: {...cronWorkflow.spec, workflowMetadata}
                                })
                            }
                        />
                    )
                }
            ]}
        />
    );
}

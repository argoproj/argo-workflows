import {Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {WorkflowTemplate} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {Loading} from '../../../shared/components/loading';
import {Status, StatusNotice} from '../../../shared/components/status-notice';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {SubmitWorkflowPanel} from '../../../workflows/components/submit-workflow-panel';
import {WorkflowTemplateEditor} from '../workflow-template-editor';

export const WorkflowTemplateDetails = (props: RouteComponentProps<any>) => {
    // boiler-plate
    const {match, location, history} = props;
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const namespace = match.params.namespace;
    const name = match.params.name;
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');
    const [tab, setTab] = useState<string>(queryParams.get('tab'));

    useEffect(
        () =>
            history.push(
                historyUrl('workflow-templates/{namespace}/{name}', {
                    namespace,
                    name,
                    sidePanel,
                    tab
                })
            ),
        [namespace, name, sidePanel, tab]
    );

    const [status, setStatus] = useState<Status>();
    const [template, setTemplate] = useState<WorkflowTemplate>();

    useEffect(() => {
        services.workflowTemplate
            .get(name, namespace)
            .then(setTemplate)
            .catch(setStatus);
    }, [name, namespace]);

    return (
        <Page
            title='Workflow Template Details'
            toolbar={{
                actionMenu: {
                    items: [
                        {
                            title: 'Submit',
                            iconClassName: 'fa fa-plus',
                            action: () => setSidePanel(true)
                        },
                        {
                            title: 'Save',
                            iconClassName: 'fa fa-save',
                            action: () =>
                                services.workflowTemplate
                                    .update(template, name, namespace)
                                    .then(setTemplate)
                                    .then(() => setStatus('Succeeded'))
                                    .catch(setStatus)
                        },
                        {
                            title: 'Delete',
                            iconClassName: 'fa fa-trash',
                            action: () => {
                                if (!confirm('Are you sure you want to delete this workflow template?\nThere is no undo.')) {
                                    return;
                                }
                                services.workflowTemplate
                                    .delete(name, namespace)
                                    .catch(setStatus)
                                    .then(() => (document.location.href = uiUrl('workflow-templates')));
                            }
                        }
                    ]
                }
            }}>
            <div className='argo-container'>
                <StatusNotice status={status} />
                {!template ? <Loading /> : <WorkflowTemplateEditor template={template} onChange={setTemplate} onError={setStatus} onTabSelected={setTab} selectedTabKey={tab} />}
            </div>
            {template && (
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)}>
                    <SubmitWorkflowPanel
                        kind='WorkflowTemplate'
                        namespace={namespace}
                        name={name}
                        entrypoint={template.spec.entrypoint}
                        entrypoints={(template.spec.templates || []).map(t => t.name)}
                        parameters={template.spec.arguments.parameters || []}
                    />
                </SlidingPanel>
            )}
        </Page>
    );
};

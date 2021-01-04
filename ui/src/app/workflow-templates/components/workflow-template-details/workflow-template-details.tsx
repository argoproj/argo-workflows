import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {WorkflowTemplate} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {SubmitWorkflowPanel} from '../../../workflows/components/submit-workflow-panel';
import {WorkflowTemplateEditor} from '../workflow-template-editor';

export const WorkflowTemplateDetails = ({history, location, match}: RouteComponentProps<any>) => {
    // boiler-plate
    const {notifications, navigation} = useContext(Context);
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

    const [error, setError] = useState<Error>();
    const [template, setTemplate] = useState<WorkflowTemplate>();
    const [edited, setEdited] = useState(false);

    useEffect(() => setEdited(true), [template]);

    useEffect(() => {
        services.workflowTemplate
            .get(name, namespace)
            .then(setTemplate)
            .then(() => setEdited(false)) // set back to false
            .then(() => setError(null))
            .catch(setError);
    }, [name, namespace]);

    return (
        <Page
            title='Workflow Template Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Workflow Templates', path: uiUrl('workflow-templates')},
                    {title: namespace, path: uiUrl('workflow-templates/' + namespace)},
                    {title: name, path: uiUrl('workflow-templates/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Submit',
                            iconClassName: 'fa fa-plus',
                            disabled: edited,
                            action: () => setSidePanel(true)
                        },
                        {
                            title: 'Update',
                            iconClassName: 'fa fa-save',
                            disabled: !edited,
                            action: () =>
                                services.workflowTemplate
                                    .update(template, name, namespace)
                                    .then(setTemplate)
                                    .then(() => notifications.show({content: 'Updated', type: NotificationType.Success}))
                                    .then(() => setEdited(false))
                                    .then(() => setError(null))
                                    .catch(setError)
                        },
                        {
                            title: 'Delete',
                            iconClassName: 'fa fa-trash',
                            disabled: edited,
                            action: () => {
                                if (!confirm('Are you sure you want to delete this workflow template?\nThere is no undo.')) {
                                    return;
                                }
                                services.workflowTemplate
                                    .delete(name, namespace)
                                    .then(() => navigation.goto(uiUrl('workflow-templates/' + namespace)))
                                    .then(() => setError(null))
                                    .catch(setError);
                            }
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!template ? <Loading /> : <WorkflowTemplateEditor template={template} onChange={setTemplate} onError={setError} onTabSelected={setTab} selectedTabKey={tab} />}
            </>
            {template && (
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)} isNarrow={true}>
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

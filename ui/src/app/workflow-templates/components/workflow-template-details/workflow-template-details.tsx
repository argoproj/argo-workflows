import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {WorkflowTemplate} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {WidgetGallery} from '../../../widgets/widget-gallery';
import {SubmitWorkflowPanel} from '../../../workflows/components/submit-workflow-panel';
import {WorkflowTemplateEditor} from '../workflow-template-editor';

export function WorkflowTemplateDetails({history, location, match}: RouteComponentProps<any>) {
    // boiler-plate
    const {notifications, navigation, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const namespace = match.params.namespace;
    const name = match.params.name;
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));
    const [tab, setTab] = useState<string>(queryParams.get('tab'));

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel'));
            setTab(p.get('tab'));
        }),
        [history]
    );

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

    useCollectEvent('openedWorkflowTemplateDetails');

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
                            action: () => setSidePanel('submit')
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
                            action: () => {
                                popup.confirm('confirm', 'Are you sure you want to delete this workflow template?').then(yes => {
                                    if (yes) {
                                        services.workflowTemplate
                                            .delete(name, namespace)
                                            .then(() => navigation.goto(uiUrl('workflow-templates/' + namespace)))
                                            .then(() => setError(null))
                                            .catch(setError);
                                    }
                                });
                            }
                        },
                        {
                            title: 'Share',
                            iconClassName: 'fa fa-share-alt',
                            action: () => setSidePanel('share')
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!template ? <Loading /> : <WorkflowTemplateEditor template={template} onChange={setTemplate} onError={setError} onTabSelected={setTab} selectedTabKey={tab} />}
            </>
            {template && (
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)} isMiddle={sidePanel === 'submit'}>
                    {sidePanel === 'submit' && (
                        <SubmitWorkflowPanel
                            kind='WorkflowTemplate'
                            namespace={namespace}
                            name={name}
                            entrypoint={template.spec.entrypoint}
                            templates={template.spec.templates || []}
                            workflowParameters={template.spec.arguments.parameters || []}
                        />
                    )}
                    {sidePanel === 'share' && <WidgetGallery namespace={namespace} label={'workflows.argoproj.io/workflow-template=' + name} />}
                </SlidingPanel>
            )}
        </Page>
    );
}

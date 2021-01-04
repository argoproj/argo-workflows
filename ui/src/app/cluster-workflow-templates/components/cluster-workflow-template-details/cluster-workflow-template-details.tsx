import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {ClusterWorkflowTemplate} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {SubmitWorkflowPanel} from '../../../workflows/components/submit-workflow-panel';
import {ClusterWorkflowTemplateEditor} from '../cluster-workflow-template-editor';

require('../../../workflows/components/workflow-details/workflow-details.scss');

export const ClusterWorkflowTemplateDetails = ({history, location, match}: RouteComponentProps<any>) => {
    // boiler-plate
    const {navigation, notifications} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const name = match.params.name;
    const [namespace, setNamespace] = useState<string>();
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');
    const [tab, setTab] = useState<string>(queryParams.get('tab'));

    const [error, setError] = useState<Error>();
    const [template, setTemplate] = useState<ClusterWorkflowTemplate>();
    const [edited, setEdited] = useState(false);

    useEffect(() => setEdited(true), [template]);
    useEffect(() => {
        history.push(historyUrl('cluster-workflow-templates/{name}', {name, sidePanel, tab}));
    }, [name, sidePanel, tab]);

    useEffect(() => {
        services.clusterWorkflowTemplate
            .get(name)
            .then(setTemplate)
            .then(() => setEdited(false)) // set back to false
            .then(() => setError(null))
            .catch(setError);
    }, [name]);

    useEffect(() => {
        services.info
            .getInfo()
            .then(info => setNamespace(Utils.getNamespace(info.managedNamespace)))
            .then(() => setError(null))
            .catch(setError);
    }, []);

    return (
        <Page
            title='Cluster Workflow Template Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Cluster Workflow Templates', path: uiUrl('cluster-workflow-templates')},
                    {title: name, path: uiUrl('cluster-workflow-templates/' + name)}
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
                            action: () => {
                                services.clusterWorkflowTemplate
                                    .update(template, name)
                                    .then(setTemplate)
                                    .then(() =>
                                        notifications.show({
                                            content: 'Updated',
                                            type: NotificationType.Success
                                        })
                                    )
                                    .then(() => setError(null))
                                    .catch(setError);
                            }
                        },
                        {
                            title: 'Delete',
                            iconClassName: 'fa fa-trash',
                            disabled: edited,
                            action: () => {
                                if (!confirm('Are you sure you want to delete this cluster workflow template?\nThere is no undo.')) {
                                    return;
                                }
                                services.clusterWorkflowTemplate
                                    .delete(name)
                                    .then(() => navigation.goto(uiUrl('cluster-workflow-templates')))
                                    .then(() => setError(null))
                                    .catch(setError);
                            }
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!template ? (
                    <Loading />
                ) : (
                    <ClusterWorkflowTemplateEditor template={template} onChange={setTemplate} onError={setError} onTabSelected={setTab} selectedTabKey={tab} />
                )}
            </>
            {template && (
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)} isNarrow={true}>
                    <SubmitWorkflowPanel
                        kind='ClusterWorkflowTemplate'
                        namespace={namespace}
                        name={template.metadata.name}
                        entrypoint={template.spec.entrypoint}
                        entrypoints={(template.spec.templates || []).map(t => t.name)}
                        parameters={template.spec.arguments.parameters || []}
                    />
                </SlidingPanel>
            )}
        </Page>
    );
};

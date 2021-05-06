import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {WorkflowTemplate} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Context} from '../../../shared/context';
import {Footnote} from '../../../shared/footnote';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {Utils} from '../../../shared/utils';
import {WorkflowTemplateCreator} from '../workflow-template-creator';

require('./workflow-template-list.scss');

const learnMore = <a href='https://argoproj.github.io/argo-workflows/workflow-templates/'>Learn more</a>;

export const WorkflowTemplateList = ({match, location, history}: RouteComponentProps<any>) => {
    // boiler-plate
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    // state for URL and query parameters
    const [namespace, setNamespace] = useState(Utils.getNamespace(match.params.namespace) || '');
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel') === 'true');
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('workflow-templates' + (Utils.managedNamespace ? '' : '/{namespace}'), {
                    namespace,
                    sidePanel
                })
            ),
        [namespace, sidePanel]
    );

    // internal state
    const [error, setError] = useState<Error>();
    const [templates, setTemplates] = useState<WorkflowTemplate[]>();

    useEffect(() => {
        services.workflowTemplate
            .list(namespace)
            .then(setTemplates)
            .then(() => setError(null))
            .catch(setError);
    }, [namespace]);

    return (
        <Page
            title='Workflow Templates'
            toolbar={{
                breadcrumbs: [
                    {title: 'Workflow Templates', path: uiUrl('workflow-templates')},
                    {title: namespace, path: uiUrl('workflow-templates/' + namespace)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Create New Workflow Template',
                            iconClassName: 'fa fa-plus',
                            action: () => setSidePanel(true)
                        }
                    ]
                },
                tools: [<NamespaceFilter key='namespace-filter' value={namespace} onChange={setNamespace} />]
            }}>
            <ErrorNotice error={error} />
            {!templates ? (
                <Loading />
            ) : templates.length === 0 ? (
                <ZeroState title='No workflow templates'>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>
                        <ExampleManifests />. {learnMore}.
                    </p>
                </ZeroState>
            ) : (
                <>
                    <div className='argo-table-list'>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-1' />
                            <div className='columns small-5'>NAME</div>
                            <div className='columns small-3'>NAMESPACE</div>
                            <div className='columns small-3'>CREATED</div>
                        </div>
                        {templates.map(t => (
                            <Link
                                className='row argo-table-list__row'
                                key={`${t.metadata.namespace}/${t.metadata.name}`}
                                to={uiUrl(`workflow-templates/${t.metadata.namespace}/${t.metadata.name}`)}>
                                <div className='columns small-1'>
                                    <i className='fa fa-clone' />
                                </div>
                                <div className='columns small-5'>{t.metadata.name}</div>
                                <div className='columns small-3'>{t.metadata.namespace}</div>
                                <div className='columns small-3'>
                                    <Timestamp date={t.metadata.creationTimestamp} />
                                </div>
                            </Link>
                        ))}
                    </div>
                    <Footnote>
                        <InfoIcon /> Workflow templates are reusable templates you can create new workflows from. <ExampleManifests />. {learnMore}.
                    </Footnote>
                </>
            )}
            <SlidingPanel isShown={sidePanel} onClose={() => setSidePanel(false)}>
                <WorkflowTemplateCreator namespace={namespace} onCreate={wf => navigation.goto(uiUrl(`workflow-templates/${wf.metadata.namespace}/${wf.metadata.name}`))} />
            </SlidingPanel>
        </Page>
    );
};

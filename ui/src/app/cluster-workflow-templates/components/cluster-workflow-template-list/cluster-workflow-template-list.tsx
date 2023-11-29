import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {ZeroState} from '../../../shared/components/zero-state';
import {Context} from '../../../shared/context';
import {useQueryParams} from '../../../shared/use-query-params';

import {Footnote} from '../../../shared/footnote';
import {services} from '../../../shared/services';
import {ClusterWorkflowTemplateCreator} from '../cluster-workflow-template-creator';

import './cluster-workflow-template-list.scss';

export function ClusterWorkflowTemplateList({history, location}: RouteComponentProps<any>) {
    const {navigation} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);
    const [templates, setTemplates] = useState<models.ClusterWorkflowTemplate[]>();
    const [error, setError] = useState<Error>();
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));

    async function fetchClusterWorkflowTemplates() {
        try {
            const retrievedTemplates = await services.clusterWorkflowTemplate.list();
            setTemplates(retrievedTemplates);
            setError(null);
        } catch (err) {
            setError(err);
        }
    }

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel'));
        }),
        [history]
    );

    useEffect(() => {
        fetchClusterWorkflowTemplates();
    }, []);

    useCollectEvent('openedClusterWorkflowTemplateList');

    function renderTemplates() {
        if (error) {
            return <ErrorNotice error={error} />;
        }
        if (!templates) {
            return <Loading />;
        }
        const learnMore = <a href='https://argoproj.github.io/argo-workflows/cluster-workflow-templates/'>Learn more</a>;
        if (templates.length === 0) {
            return (
                <ZeroState title='No cluster workflow templates'>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>
                        <ExampleManifests />. {learnMore}.
                    </p>
                </ZeroState>
            );
        }
        return (
            <>
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1' />
                        <div className='columns small-5'>NAME</div>
                        <div className='columns small-3'>CREATED</div>
                    </div>
                    {templates.map(t => (
                        <Link className='row argo-table-list__row' key={t.metadata.uid} to={uiUrl(`cluster-workflow-templates/${t.metadata.name}`)}>
                            <div className='columns small-1'>
                                <i className='fa fa-clone' />
                            </div>
                            <div className='columns small-5'>{t.metadata.name}</div>
                            <div className='columns small-3'>
                                <Timestamp date={t.metadata.creationTimestamp} />
                            </div>
                        </Link>
                    ))}
                </div>
                <Footnote>
                    <InfoIcon /> Cluster scoped Workflow templates are reusable templates you can create new workflows from. <ExampleManifests />. {learnMore}.
                </Footnote>
            </>
        );
    }

    return (
        <Page
            title='Cluster Workflow Templates'
            toolbar={{
                breadcrumbs: [{title: 'Cluster Workflow Templates', path: uiUrl('cluster-workflow-templates')}],
                actionMenu: {
                    items: [
                        {
                            title: 'Create New Cluster Workflow Template',
                            iconClassName: 'fa fa-plus',
                            action: () => setSidePanel('new')
                        }
                    ]
                }
            }}>
            {renderTemplates()}
            <SlidingPanel isShown={sidePanel !== null} onClose={() => setSidePanel(null)}>
                <ClusterWorkflowTemplateCreator onCreate={wf => navigation.goto(uiUrl(`cluster-workflow-templates/${wf.metadata.name}`))} />
            </SlidingPanel>
        </Page>
    );
}

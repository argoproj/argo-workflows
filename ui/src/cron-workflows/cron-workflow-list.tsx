import {Page} from 'argo-ui/src/components/page/page';
import {SlidingPanel} from 'argo-ui/src/components/sliding-panel/sliding-panel';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router-dom';

import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {ExampleManifests} from '../shared/components/example-manifests';
import {InfoIcon} from '../shared/components/fa-icons';
import {Loading} from '../shared/components/loading';
import {TimestampSwitch} from '../shared/components/timestamp';
import {ZeroState} from '../shared/components/zero-state';
import {Context} from '../shared/context';
import {Footnote} from '../shared/footnote';
import {historyUrl} from '../shared/history';
import {CronWorkflow} from '../shared/models';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {useCollectEvent} from '../shared/use-collect-event';
import {useQueryParams} from '../shared/use-query-params';
import useTimestamp, {TIMESTAMP_KEYS} from '../shared/use-timestamp';
import {CronWorkflowCreator} from './cron-workflow-creator';
import {CronWorkflowFilters} from './cron-workflow-filters';
import {CronWorkflowRow} from './cron-workflow-row';

import './cron-workflow-list.scss';

const learnMore = <a href='https://argo-workflows.readthedocs.io/en/latest/cron-workflows/'>Learn more</a>;

export function CronWorkflowList({match, location, history}: RouteComponentProps<any>) {
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    // state for URL, query, and label parameters
    const [namespace, setNamespace] = useState<string>(nsUtils.getNamespace(match.params.namespace) || '');
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');
    const [labels, setLabels] = useState<string[]>([]);
    const [states, setStates] = useState(['Running', 'Suspended']); // check all by default

    const [storedDisplayISOFormatCreation, setStoredDisplayISOFormatCreation] = useTimestamp(TIMESTAMP_KEYS.CRON_WORKFLOW_LIST_CREATION);
    const [storedDisplayISOFormatNextScheduled, setStoredDisplayISOFormatNextScheduled] = useTimestamp(TIMESTAMP_KEYS.CRON_WORKFLOW_LIST_NEXT_SCHEDULED);

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel') === 'true');
        }),
        [history]
    );

    // save history
    useEffect(
        () =>
            history.push(
                historyUrl('cron-workflows' + (nsUtils.getManagedNamespace() ? '' : '/{namespace}'), {
                    namespace,
                    sidePanel
                })
            ),
        [namespace, sidePanel]
    );

    // internal state
    const [error, setError] = useState<Error>();
    const [cronWorkflows, setCronWorkflows] = useState<CronWorkflow[]>();

    useEffect(() => {
        (async () => {
            try {
                const list = await services.cronWorkflows.list(namespace, labels);
                if (states.length === 1) {
                    if (states.includes('Suspended')) {
                        setCronWorkflows(list.filter(el => el.spec.suspend === true));
                    } else {
                        setCronWorkflows(list.filter(el => el.spec.suspend !== true));
                    }
                } else {
                    setCronWorkflows(list);
                }
                setError(null);
            } catch (newError) {
                setError(newError);
            }
        })();
    }, [namespace, labels.toString(), states.toString()]); // referential equality, so use values, not refs

    useCollectEvent('openedCronWorkflowList');

    return (
        <Page
            title='Cron Workflows'
            toolbar={{
                breadcrumbs: [
                    {title: 'Cron Workflows', path: uiUrl('cron-workflows')},
                    {title: namespace, path: uiUrl('cron-workflows/' + namespace)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Create New Cron Workflow',
                            iconClassName: 'fa fa-plus',
                            action: () => setSidePanel(true)
                        }
                    ]
                }
            }}>
            <div className='row'>
                <div className='columns small-12 xlarge-2'>
                    <div>
                        <CronWorkflowFilters
                            cronWorkflows={cronWorkflows || []}
                            namespace={namespace}
                            labels={labels}
                            states={states}
                            onChange={(namespaceValue: string, labelsValue: string[], stateValue: string[]) => {
                                setNamespace(namespaceValue);
                                setLabels(labelsValue);
                                setStates(stateValue);
                            }}
                        />
                    </div>
                </div>
                <div className='columns small-12 xlarge-10'>
                    <ErrorNotice error={error} />
                    {!cronWorkflows ? (
                        <Loading />
                    ) : cronWorkflows.length === 0 ? (
                        <ZeroState title='No cron workflows'>
                            <p>You can create new cron workflows here or using the CLI.</p>
                            <p>
                                <ExampleManifests />. {learnMore}.
                            </p>
                        </ZeroState>
                    ) : (
                        <>
                            <div className='argo-table-list'>
                                <div className='row argo-table-list__head'>
                                    <div className='columns small-1' />
                                    <div className='columns small-2'>NAME</div>
                                    <div className='columns small-2'>NAMESPACE</div>
                                    <div className='columns small-1'>TimeZone</div>
                                    <div className='columns small-1'>SCHEDULES</div>
                                    <div className='columns small-1' />
                                    <div className='columns small-2'>
                                        CREATED{' '}
                                        <TimestampSwitch storedDisplayISOFormat={storedDisplayISOFormatCreation} setStoredDisplayISOFormat={setStoredDisplayISOFormatCreation} />
                                    </div>
                                    <div className='columns small-2'>
                                        NEXT RUN{' '}
                                        <TimestampSwitch
                                            storedDisplayISOFormat={storedDisplayISOFormatNextScheduled}
                                            setStoredDisplayISOFormat={setStoredDisplayISOFormatNextScheduled}
                                        />
                                    </div>
                                </div>
                                {cronWorkflows.map(w => {
                                    return (
                                        <CronWorkflowRow
                                            workflow={w}
                                            displayISOFormatCreation={storedDisplayISOFormatCreation}
                                            displayISOFormatNextScheduled={storedDisplayISOFormatNextScheduled}
                                            key={`{w.metadata.namespace}/${w.metadata.name}`}
                                        />
                                    );
                                })}
                            </div>
                            <Footnote>
                                <InfoIcon /> Cron workflows are workflows that run on a preset schedule. Next scheduled run assumes workflow-controller is in UTC.{' '}
                                <ExampleManifests />. {learnMore}.
                            </Footnote>
                        </>
                    )}
                </div>
            </div>
            <SlidingPanel isShown={sidePanel} onClose={() => setSidePanel(false)}>
                <CronWorkflowCreator namespace={namespace} onCreate={wf => navigation.goto(uiUrl(`cron-workflows/${wf.metadata.namespace}/${wf.metadata.name}`))} />
            </SlidingPanel>
        </Page>
    );
}

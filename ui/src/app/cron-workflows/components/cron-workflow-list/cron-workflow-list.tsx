import {Page, SlidingPanel, Ticker} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {CronWorkflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {Loading} from '../../../shared/components/loading';
import {Timestamp} from '../../../shared/components/timestamp';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {ZeroState} from '../../../shared/components/zero-state';
import {Context} from '../../../shared/context';
import {getNextScheduledTime} from '../../../shared/cron';
import {Footnote} from '../../../shared/footnote';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {Utils} from '../../../shared/utils';
import {CronWorkflowCreator} from '../cron-workflow-creator';
import {CronWorkflowFilters} from '../cron-workflow-filters/cron-workflow-filters';
import {PrettySchedule} from '../pretty-schedule';

import './cron-workflow-list.scss';

const learnMore = <a href='https://argoproj.github.io/argo-workflows/cron-workflows/'>Learn more</a>;

export function CronWorkflowList({match, location, history}: RouteComponentProps<any>) {
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    // state for URL, query, and label parameters
    const [namespace, setNamespace] = useState<string>(Utils.getNamespace(match.params.namespace) || '');
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');
    const [labels, setLabels] = useState<string[]>([]);
    const [states, setStates] = useState(['Running', 'Suspended']); // check all by default

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
                historyUrl('cron-workflows' + (Utils.managedNamespace ? '' : '/{namespace}'), {
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
                                    <div className='columns small-3'>NAME</div>
                                    <div className='columns small-2'>NAMESPACE</div>
                                    <div className='columns small-1'>TimeZone</div>
                                    <div className='columns small-1'>SCHEDULE</div>
                                    <div className='columns small-2' />
                                    <div className='columns small-1'>CREATED</div>
                                    <div className='columns small-1'>NEXT RUN</div>
                                </div>
                                {cronWorkflows.map(w => (
                                    <Link
                                        className='row argo-table-list__row'
                                        key={`${w.metadata.namespace}/${w.metadata.name}`}
                                        to={uiUrl(`cron-workflows/${w.metadata.namespace}/${w.metadata.name}`)}>
                                        <div className='columns small-1'>{w.spec.suspend ? <i className='fa fa-pause' /> : <i className='fa fa-clock' />}</div>
                                        <div className='columns small-3'>{w.metadata.name}</div>
                                        <div className='columns small-2'>{w.metadata.namespace}</div>
                                        <div className='columns small-1'>{w.spec.timezone}</div>
                                        <div className='columns small-1'>{w.spec.schedule}</div>
                                        <div className='columns small-2'>
                                            <PrettySchedule schedule={w.spec.schedule} />
                                        </div>
                                        <div className='columns small-1'>
                                            <Timestamp date={w.metadata.creationTimestamp} />
                                        </div>
                                        <div className='columns small-1'>
                                            {w.spec.suspend ? (
                                                ''
                                            ) : (
                                                <Ticker intervalMs={1000}>{() => <Timestamp date={getNextScheduledTime(w.spec.schedule, w.spec.timezone)} />}</Ticker>
                                            )}
                                        </div>
                                    </Link>
                                ))}
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

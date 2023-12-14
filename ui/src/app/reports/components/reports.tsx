import {Page} from 'argo-ui/src/index';
import {ChartOptions} from 'chart.js';
import 'chartjs-plugin-annotation';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {Bar, ChartData} from 'react-chartjs-2';
import {RouteComponentProps} from 'react-router-dom';
import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {InfoIcon} from '../../shared/components/fa-icons';
import {useCollectEvent} from '../../shared/components/use-collect-event';
import {ZeroState} from '../../shared/components/zero-state';
import {Context} from '../../shared/context';
import {Footnote} from '../../shared/footnote';
import {historyUrl} from '../../shared/history';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';
import {ReportFilters} from './reports-filters';
import {workflowsToChartData} from './workflows-to-chart-data';

interface Chart {
    data: ChartData<any>;
    options: ChartOptions;
}

const limit = 100;

export function Reports({match, location, history}: RouteComponentProps<any>) {
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    // state for URL, query, and label parameters
    const [namespace, setNamespace] = useState<string>(Utils.getNamespace(match.params.namespace) || '');
    const [labels, setLabels] = useState((queryParams.get('labels') || '').split(',').filter(v => v !== ''));
    // internal state
    const [charts, setCharts] = useState<Chart[]>();
    const [error, setError] = useState<Error>();

    // save history
    useEffect(() => {
        history.push(historyUrl('reports' + (Utils.managedNamespace ? '' : '/{namespace}'), {namespace, labels: labels.join(',')}));
    }, [namespace, labels]);

    async function onChange(newNamespace: string, newLabels: string[]) {
        if (newNamespace === '' || newLabels.length === 0) {
            setNamespace(newNamespace);
            setLabels(newLabels);
            setCharts(null);
            return;
        }
        setNamespace(newNamespace);
        setLabels(newLabels);
    }

    useEffect(() => {
        (async () => {
            try {
                const list = await services.workflows.list(namespace, [], labels, {limit}, [
                    'items.metadata.name',
                    'items.status.phase',
                    'items.status.startedAt',
                    'items.status.finishedAt',
                    'items.status.resourcesDuration'
                ]);
                const newCharts = workflowsToChartData(list.items || [], limit);
                setCharts(newCharts);
                setError(null);
            } catch (newError) {
                setError(newError);
            }
        })();
    }, [namespace, labels.toString()]); // referential equality, so use values, not refs

    useCollectEvent('openedReports');

    return (
        <Page
            title='Reports'
            toolbar={{
                breadcrumbs: [
                    {title: 'Reports', path: uiUrl('reports')},
                    {title: namespace, path: uiUrl('reports/' + namespace)}
                ]
            }}>
            <div className='row'>
                <div className='columns small-12 xlarge-2'>
                    <ReportFilters namespace={namespace} labels={labels} onChange={onChange} />
                </div>
                <div className='columns small-12 xlarge-10'>
                    <ErrorNotice error={error} />
                    {!charts ? (
                        <ZeroState title='Workflow Report'>
                            <p>
                                Use this page to find costly or time consuming workflows. You must label workflows you want to report on. If you use <b>workflow templates</b> or{' '}
                                <b>cron workflows</b>, your workflows will be automatically labelled. You&apos;ll probably need to enable the{' '}
                                <a href='https://argoproj.github.io/argo-workflows/workflow-archive/'>workflow archive</a> to get long term data. Only the {limit} most recent
                                workflows are shown.
                            </p>
                            <p>Select a namespace and at least one label to get a report.</p>
                            <p>
                                <a href='https://argoproj.github.io/argo-workflows/cost-optimisation/'>Learn more about cost optimization</a>
                            </p>
                        </ZeroState>
                    ) : (
                        <>
                            {charts.map(chart => (
                                <div key={chart.data.name}>
                                    <div className='white-box'>
                                        <Bar
                                            data={chart.data}
                                            options={chart.options}
                                            onElementsClick={(e: any[]) => {
                                                const activePoint = e[0];
                                                if (activePoint === undefined) {
                                                    return;
                                                }
                                                const workflowName = chart.data.labels[activePoint._index];
                                                navigation.goto(uiUrl('workflows/' + namespace + '/' + workflowName));
                                            }}
                                        />
                                    </div>
                                </div>
                            ))}
                            <Footnote>
                                <InfoIcon /> {charts[0].data.labels.length} records.
                            </Footnote>
                        </>
                    )}
                </div>
            </div>
        </Page>
    );
}

export default Reports;

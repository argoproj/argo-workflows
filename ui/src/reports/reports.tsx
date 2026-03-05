import {Page} from 'argo-ui/src/components/page/page';
import {ChartOptions} from 'chart.js';

import 'chartjs-plugin-annotation';

import * as React from 'react';
import {useContext, useEffect, useRef, useState} from 'react';
import {Bar, ChartData} from 'react-chartjs-2';
import {RouteComponentProps} from 'react-router-dom';

import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {InfoIcon} from '../shared/components/fa-icons';
import {ZeroState} from '../shared/components/zero-state';
import {Context} from '../shared/context';
import {Footnote} from '../shared/footnote';
import {historyUrl} from '../shared/history';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {useThemeContext} from '../shared/theme-context';
import {useCollectEvent} from '../shared/use-collect-event';
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
    const {resolvedTheme} = useThemeContext();
    const chartFontColor = resolvedTheme === 'dark' ? '#e0e0e0' : '#666';

    // state for URL, query, and label parameters
    const isFirstRender = useRef(true);
    const [namespace, setNamespace] = useState<string>(nsUtils.getNamespace(match.params.namespace) || '');
    const [labels, setLabels] = useState((queryParams.get('labels') || '').split(',').filter(v => v !== ''));
    // internal state
    const [charts, setCharts] = useState<Chart[]>();
    const [error, setError] = useState<Error>();

    // save history
    useEffect(() => {
        if (isFirstRender.current) {
            isFirstRender.current = false;
            return;
        }
        history.push(historyUrl('reports' + (nsUtils.getManagedNamespace() ? '' : '/{namespace}'), {namespace, labels: labels.join(',')}));
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
                const newCharts = workflowsToChartData(list.items || [], limit, chartFontColor);
                setCharts(newCharts);
                setError(null);
            } catch (newError) {
                setError(newError);
            }
        })();
    }, [namespace, labels.toString(), chartFontColor]); // referential equality, so use values, not refs

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
                                <a href='https://argo-workflows.readthedocs.io/en/latest/workflow-archive/'>workflow archive</a> to get long term data. Only the {limit} most recent
                                workflows are shown.
                            </p>
                            <p>Select a namespace and at least one label to get a report.</p>
                            <p>
                                <a href='https://argo-workflows.readthedocs.io/en/latest/cost-optimisation/'>Learn more about cost optimization</a>
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

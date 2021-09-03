import {SlidingPanel, Tabs} from 'argo-ui';
import * as React from 'react';
import {Step, StepStatus} from '../../../models/step';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';
import {Phase} from '../../shared/components/phase';
import {TickMeter} from '../../shared/components/tick-meter';
import {Timestamp} from '../../shared/components/timestamp';
import {parseResourceQuantity} from '../../shared/resource-quantity';
import {EventsPanel} from '../../workflows/components/events-panel';
import {PipelineLogsViewer} from './pipeline-logs-viewer';
import {totalRate} from './total-rate';

const prettyNumber = (x: number): number => (x < 1 ? x : Math.round(x));

export const StepSidePanel = ({
    isShown,
    namespace,
    pipelineName,
    step,
    setTab,
    tab,
    onClose
}: {
    isShown: boolean;
    namespace: string;
    pipelineName: string;
    step: Step;
    tab: string;
    setTab: (tab: string) => void;
    onClose: () => void;
}) => {
    const stepName = step.spec.name;
    return (
        <SlidingPanel isShown={isShown} onClose={onClose}>
            <>
                <h4>
                    {pipelineName}/{stepName}
                </h4>
                <Tabs
                    navTransparent={true}
                    selectedTabKey={tab}
                    onTabSelected={setTab}
                    tabs={[
                        {
                            title: 'STATUS',
                            key: 'status',
                            content: statusPanel(step)
                        },
                        {
                            title: 'LOGS',
                            key: 'logs',
                            content: <PipelineLogsViewer namespace={namespace} pipelineName={pipelineName} stepName={stepName} />
                        },
                        {
                            title: 'EVENTS',
                            key: 'events',
                            content: <EventsPanel kind='Step' namespace={namespace} name={step.metadata.name} />
                        },
                        {
                            title: 'MANIFEST',
                            key: 'manifest',
                            content: <ObjectEditor value={step} />
                        }
                    ]}
                />
            </>
        </SlidingPanel>
    );
};

const statusPanel = (step: Step) =>
    step.status && (
        <>
            <div className='row'>
                <div className='columns small-12'>{statusHeader(step.status)}</div>
            </div>
            <div className='row' style={{marginTop: 10}}>
                <div className='columns small-6'>{sourcesPanel(step.status)}</div>
                <div className='columns small-6'>{sinksPanel(step.status)}</div>
            </div>
        </>
    );

const statusHeader = (status: StepStatus) => (
    <div className='white-box'>
        <div className='white-box__details'>
            <div className='row white-box__details-row'>
                <div className='columns small-3'>Phase</div>
                <div className='columns small-9'>
                    <Phase value={status.phase} /> {status.message}
                </div>
            </div>
            <div className='row white-box__details-row'>
                <div className='columns small-3'>Replicas</div>
                <div className='columns small-3'>{status.replicas}</div>
                {status.lastScaledAt && (
                    <>
                        <div className='columns small-3'>Last scaled</div>
                        <div className='columns small-3'>
                            <Timestamp date={status.lastScaledAt} />
                        </div>
                    </>
                )}
            </div>
        </div>
    </div>
);

const sourcesPanel = (status: StepStatus) => (
    <>
        <h5>Sources</h5>
        {status.sourceStatuses ? (
            Object.entries(status.sourceStatuses).map(([name, x]) => {
                const total = Object.values(x.metrics || {})
                    .filter(m => m.total)
                    .reduce((a, b) => a + b.total, 0);
                const rate = totalRate(x.metrics, status.replicas);
                const errors = Object.values(x.metrics || {})
                    .filter(m => m.errors)
                    .reduce((a, b) => a + b.errors, 0);
                const retries = Object.values(x.metrics || {})
                    .filter(m => m.retries)
                    .reduce((a, b) => a + b.retries, 0);
                return (
                    <div className='white-box' key={name}>
                        <p>{name}</p>
                        <div className='white-box__details'>
                            <div className='row white-box__details-row'>
                                <div className='columns small-4'>Pending</div>
                                <div className='columns small-8'>
                                    <TickMeter value={x.pending || 0} />
                                </div>
                            </div>
                            <div className='row white-box__details-row'>
                                <div className='columns small-4'>Retries</div>
                                <div className='columns small-8'>
                                    <TickMeter value={retries} />
                                </div>
                            </div>
                            <div className='row white-box__details-row'>
                                <div className='columns small-4'>Total</div>
                                <div className='columns small-4'>
                                    <TickMeter value={total} />
                                </div>
                                <div className='columns small-4' title='Rate'>
                                    <TickMeter value={rate} /> <small>TPS</small>
                                </div>
                            </div>
                            <div className='row white-box__details-row'>
                                <div className='columns small-4'>Errors</div>
                                <div className='columns small-4'>
                                    <TickMeter value={errors} />
                                </div>
                                <div className='columns small-4'>
                                    <TickMeter value={Math.floor((10000 * errors) / total) / 100} />%
                                </div>
                            </div>
                        </div>
                    </div>
                );
            })
        ) : (
            <div className='white-box'>None</div>
        )}
    </>
);

const sinksPanel = (status: StepStatus) => (
    <>
        <h5>Sinks</h5>
        {status.sinkStatuses ? (
            Object.entries(status.sinkStatuses).map(([name, x]) => {
                const total = Object.values(x.metrics || {})
                    .filter(m => m.total)
                    .reduce((a, b) => a + b.total, 0);
                const rate = Object.entries(x.metrics || {})
                    // the rate will remain after scale-down, so we must filter out, as it'll be wrong
                    .filter(([replica]) => parseInt(replica, 10) < status.replicas)
                    .map(([, m]) => m)
                    .map(m => parseResourceQuantity(m.rate))
                    .reduce((a, b) => a + b, 0);
                const errors = Object.values(x.metrics || {})
                    .filter(m => m.errors)
                    .reduce((a, b) => a + b.errors, 0);
                return (
                    <div className='white-box' key={name}>
                        <p>{name}</p>
                        <div className='white-box__details'>
                            <div className='row white-box__details-row'>
                                <div className='columns small-4'>Total</div>
                                <div className='columns small-4'>
                                    <TickMeter value={total} />
                                </div>
                                <div className='columns small-4' title='Rate'>
                                    <TickMeter value={prettyNumber(rate)} /> <small>TPS</small>
                                </div>
                            </div>
                            <div className='row white-box__details-row'>
                                <div className='columns small-4'>Errors</div>
                                <div className='columns small-4'>
                                    <TickMeter value={errors} />
                                </div>
                                <div className='columns small-4'>
                                    <TickMeter value={prettyNumber(Math.floor((10000 * errors) / total) / 100)} />%
                                </div>
                            </div>
                        </div>
                    </div>
                );
            })
        ) : (
            <div className='white-box'>None</div>
        )}
    </>
);

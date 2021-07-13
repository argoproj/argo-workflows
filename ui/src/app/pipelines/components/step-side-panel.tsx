import {SlidingPanel, Tabs} from 'argo-ui';
import * as React from 'react';
import {Step} from '../../../models/step';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';
import {Phase} from '../../shared/components/phase';
import {TickMeter} from '../../shared/components/tick-meter';
import {Timestamp} from '../../shared/components/timestamp';
import {parseResourceQuantity} from '../../shared/resource-quantity';
import {EventsPanel} from '../../workflows/components/events-panel';
import {PipelineLogsViewer} from './pipeline-logs-viewer';

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
                            content: step.status && (
                                <>
                                    <div className='white-box'>
                                        <div className='white-box__details'>
                                            <div className='row white-box__details-row'>
                                                <div className='columns small-3'>Phase</div>
                                                <div className='columns small-9'>
                                                    <Phase value={step.status.phase} /> {step.status.message}
                                                </div>
                                            </div>
                                            <div className='row white-box__details-row'>
                                                <div className='columns small-3'>Replicas</div>
                                                <div className='columns small-3'>{step.status.replicas}</div>
                                                {step.status.lastScaledAt && (
                                                    <>
                                                        <div className='columns small-3'>Last scaled</div>
                                                        <div className='columns small-3'>
                                                            <Timestamp date={step.status.lastScaledAt} />
                                                        </div>
                                                    </>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                    <h5>Source Statuses</h5>
                                    {step.status.sourceStatuses ? (
                                        Object.entries(step.status.sourceStatuses).map(([name, x]) => {
                                            const total = Object.values(x.metrics || {})
                                                .filter(m => m.total)
                                                .reduce((a, b) => a + b.total, 0);
                                            const rate = Object.entries(x.metrics || {})
                                                // the rate will remain after scale-down, so we must filter out, as it'll be wrong
                                                .filter(([replica, m]) => parseInt(replica, 10) < step.status.replicas)
                                                .map(([, m]) => m)
                                                .map(m => parseResourceQuantity(m.rate))
                                                .reduce((a, b) => a + b, 0);
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
                                                            <div className='columns small-2'>Pending</div>
                                                            <div className='columns small-4'>
                                                                <TickMeter value={x.pending || 0} />
                                                            </div>

                                                            <div className='columns small-2'>Retries</div>
                                                            <div className='columns small-4'>
                                                                <TickMeter value={retries} />
                                                            </div>
                                                        </div>
                                                        <div className='row white-box__details-row'>
                                                            <div className='columns small-2'>Total</div>
                                                            <div className='columns small-2'>
                                                                <TickMeter value={total} />
                                                            </div>
                                                            <div className='columns small-2' title='Rate'>
                                                                ＊<TickMeter value={rate} /> <small>TPS</small>
                                                            </div>
                                                            <div className='columns small-5'>{x.lastMessage ? x.lastMessage.data : '-'}</div>
                                                            <div className='columns small-1'>{x.lastMessage ? <Timestamp date={x.lastMessage.time} /> : '-'}</div>
                                                        </div>
                                                        <div className='row white-box__details-row'>
                                                            <div className='columns small-2'>Errors</div>
                                                            <div className='columns small-2'>
                                                                <TickMeter value={errors} />
                                                            </div>
                                                            <div className='columns small-2'>
                                                                <TickMeter value={Math.floor((10000 * errors) / total) / 100} />%
                                                            </div>
                                                            <div className='columns small-5'>{x.lastError ? x.lastError.message : '-'}</div>
                                                            <div className='columns small-1'>{x.lastError ? <Timestamp date={x.lastError.time} /> : '-'}</div>
                                                        </div>
                                                    </div>
                                                </div>
                                            );
                                        })
                                    ) : (
                                        <div className='white-box'>None</div>
                                    )}
                                    <h5>Sink Statues</h5>
                                    {step.status.sinkStatuses ? (
                                        Object.entries(step.status.sinkStatuses).map(([name, x]) => {
                                            const total = Object.values(x.metrics || {})
                                                .filter(m => m.total)
                                                .reduce((a, b) => a + b.total, 0);
                                            const rate = Object.entries(x.metrics || {})
                                                // the rate will remain after scale-down, so we must filter out, as it'll be wrong
                                                .filter(([replica, m]) => parseInt(replica, 10) < step.status.replicas)
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
                                                            <div className='columns small-2'>Total</div>
                                                            <div className='columns small-2'>
                                                                <TickMeter value={total} />
                                                            </div>

                                                            <div className='columns small-2' title='Rate'>
                                                                ＊<TickMeter value={rate} /> <small>TPS</small>
                                                            </div>
                                                            <div className='columns small-5'>{x.lastMessage ? x.lastMessage.data : '-'}</div>
                                                            <div className='columns small-1'>{x.lastMessage ? <Timestamp date={x.lastMessage.time} /> : '-'}</div>
                                                        </div>
                                                        <div className='row white-box__details-row'>
                                                            <div className='columns small-2'>Errors</div>
                                                            <div className='columns small-2'>
                                                                <TickMeter value={errors} />
                                                            </div>
                                                            <div className='columns small-2'>
                                                                <TickMeter value={Math.floor((10000 * errors) / total) / 100} />%
                                                            </div>
                                                            <div className='columns small-5'>{x.lastError ? x.lastError.message : '-'}</div>
                                                            <div className='columns small-1'>{x.lastError ? <Timestamp date={x.lastError.time} /> : '-'}</div>
                                                        </div>
                                                    </div>
                                                </div>
                                            );
                                        })
                                    ) : (
                                        <div className='white-box'>None</div>
                                    )}
                                </>
                            )
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

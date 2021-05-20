import {SlidingPanel, Tabs} from 'argo-ui';
import * as React from 'react';
import {Step} from '../../../models/step';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';
import {Phase} from '../../shared/components/phase';
import {Timestamp} from '../../shared/components/timestamp';
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
                                        Object.entries(step.status.sourceStatuses).map(([name, x]) => (
                                            <div className='white-box'>
                                                <p>{name}</p>
                                                <div className='white-box__details'>
                                                    <div className='row white-box__details-row'>
                                                        <div className='columns small-2'>Pending</div>
                                                        <div className='columns small-10'>{x.pending || '-'}</div>
                                                    </div>
                                                    <div className='row white-box__details-row'>
                                                        <div className='columns small-2'>Message</div>
                                                        <div className='columns small-1'>{Object.values(x.metrics || {}).reduce((a, b) => a + b.total || 0, 0)}</div>
                                                        <div className='columns small-6'>{x.lastMessage ? x.lastMessage.data : '-'}</div>
                                                        <div className='columns small-3'>{x.lastMessage ? <Timestamp date={x.lastMessage.time} /> : '-'}</div>
                                                    </div>
                                                    <div className='row white-box__details-row'>
                                                        <div className='columns small-2'>Errors</div>
                                                        <div className='columns small-1'>{Object.values(x.metrics || {}).reduce((a, b) => a + b.errors || 0, 0)}</div>
                                                        <div className='columns small-6'>{x.lastError ? x.lastError.message : '-'}</div>
                                                        <div className='columns small-3'>{x.lastError ? <Timestamp date={x.lastError.time} /> : '-'}</div>
                                                    </div>
                                                </div>
                                            </div>
                                        ))
                                    ) : (
                                        <div className='white-box'>None</div>
                                    )}
                                    <h5>Sink Statues</h5>
                                    {step.status.sinkStatuses ? (
                                        Object.entries(step.status.sinkStatuses).map(([name, x]) => (
                                            <div className='white-box'>
                                                <p>{name}</p>
                                                <div className='white-box__details'>
                                                    <div className='row white-box__details-row'>
                                                        <div className='columns small-2'>Message</div>
                                                        <div className='columns small-1'>{Object.values(x.metrics || {}).reduce((a, b) => a + b.total || 0, 0)}</div>
                                                        <div className='columns small-6'>{x.lastMessage ? x.lastMessage.data : '-'}</div>
                                                        <div className='columns small-3'>{x.lastMessage ? <Timestamp date={x.lastMessage.time} /> : '-'}</div>
                                                    </div>
                                                    <div className='row white-box__details-row'>
                                                        <div className='columns small-2'>Errors</div>
                                                        <div className='columns small-1'>{Object.values(x.metrics || {}).reduce((a, b) => a + b.errors || 0, 0)}</div>
                                                        <div className='columns small-6'>{x.lastError ? x.lastError.message : '-'}</div>
                                                        <div className='columns small-3'>{x.lastError ? <Timestamp date={x.lastError.time} /> : '-'}</div>
                                                    </div>
                                                </div>
                                            </div>
                                        ))
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

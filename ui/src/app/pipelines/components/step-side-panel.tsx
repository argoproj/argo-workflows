import {SlidingPanel, Tabs} from 'argo-ui';
import * as React from 'react';
import {Step, StepStatus} from '../../../models/step';
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

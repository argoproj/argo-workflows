import {SlidingPanel, Tabs} from 'argo-ui';
import * as React from 'react';
import {Step} from '../../../models/step';
import {Notice} from '../../shared/components/notice';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';
import {Phase} from '../../shared/components/phase';
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
                            content: (
                                <>
                                    <Notice style={{marginLeft: 0, marginRight: 0}}>
                                        <Phase value={(step.status || {}).phase} /> {(step.status || {}).message}
                                    </Notice>
                                    <ObjectEditor value={step.status} />
                                </>
                            )
                        },
                        {
                            title: 'SPEC',
                            key: 'spec',
                            content: <ObjectEditor value={step.spec} />
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
                        }
                    ]}
                />
            </>
        </SlidingPanel>
    );
};

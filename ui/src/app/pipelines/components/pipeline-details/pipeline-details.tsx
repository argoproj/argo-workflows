import {NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {Pipeline} from '../../../../models/pipeline';
import {Step} from '../../../../models/step';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Loading} from '../../../shared/components/loading';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {ListWatch} from '../../../shared/list-watch';
import {services} from '../../../shared/services';
import {StepSidePanel} from '../step-side-panel';
import {graph} from './pipeline-graph';

require('./pipeline.scss');

export const PipelineDetails = ({history, match, location}: RouteComponentProps<any>) => {
    const {notifications, navigation, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);
    // state for URL and query parameters
    const namespace = match.params.namespace;
    const name = match.params.name;

    const [tab, setTab] = useState(queryParams.get('tab'));
    const [selectedStep, selectStep] = useState(queryParams.get('selectedStep'));

    useEffect(
        () =>
            history.push(
                historyUrl('pipelines/{namespace}/{name}', {
                    namespace,
                    name,
                    selectedStep,
                    tab
                })
            ),
        [namespace, name, selectedStep, tab]
    );

    const [error, setError] = useState<Error>();
    const [pipeline, setPipeline] = useState<Pipeline>();
    const [steps, setSteps] = useState<Step[]>([]);

    useEffect(() => {
        services.pipeline
            .getPipeline(namespace, name)
            .then(setPipeline)
            .then(() => setError(null))
            .catch(setError);
        const w = new ListWatch<Step>(
            () => Promise.resolve({metadata: {}, items: []}),
            () => services.pipeline.watchSteps(namespace, ['dataflow.argoproj.io/pipeline-name=' + name]),
            () => setError(null),
            () => setError(null),
            items => setSteps([...items]),
            setError
        );
        w.start();
        return () => w.stop();
    }, [name, namespace]);

    const step = steps.find(s => s.spec.name === selectedStep);
    return (
        <Page
            title='Pipeline Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Pipelines', path: uiUrl('pipelines')},
                    {title: namespace, path: uiUrl('pipelines/' + namespace)},
                    {title: name, path: uiUrl('pipelines/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Restart',
                            iconClassName: 'fa fa-redo',
                            action: () => {
                                services.pipeline
                                    .restartPipeline(namespace, name)
                                    .then(() => setError(null))
                                    .then(() =>
                                        notifications.show({type: NotificationType.Success, content: 'Your pipeline pods should terminate within ~30s, before being re-created'})
                                    )
                                    .catch(setError);
                            }
                        },
                        {
                            title: 'Delete',
                            iconClassName: 'fa fa-trash',
                            action: () => {
                                popup.confirm('confirm', 'Are you sure you want to delete this pipeline?').then(yes => {
                                    if (yes) {
                                        services.pipeline
                                            .deletePipeline(namespace, name)
                                            .then(() => navigation.goto(uiUrl('pipelines/' + namespace)))
                                            .then(() => setError(null))
                                            .catch(setError);
                                    }
                                });
                            }
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!pipeline ? (
                    <Loading />
                ) : (
                    <>
                        <GraphPanel
                            storageScope='pipeline'
                            classNames='pipeline'
                            graph={graph(pipeline, steps)}
                            nodeGenresTitle='Type'
                            nodeGenres={{
                                cat: true,
                                code: true,
                                container: true,
                                cron: true,
                                db: true,
                                dedupe: true,
                                expand: true,
                                filter: true,
                                flatten: true,
                                git: true,
                                group: true,
                                http: true,
                                kafka: true,
                                log: true,
                                map: true,
                                s3: true,
                                stan: true,
                                unknown: true,
                                volume: true
                            }}
                            defaultIconShape='circle'
                            nodeClassNamesTitle='Status'
                            nodeClassNames={{
                                '': true,
                                'Pending': true,
                                'Running': true,
                                'Failed': true,
                                'Succeeded': true
                            }}
                            horizontal={true}
                            selectedNode={selectedStep}
                            onNodeSelect={node => {
                                if (node.startsWith('step/')) {
                                    selectStep(node.replace('step/', ''));
                                }
                            }}
                        />
                        {step && (
                            <StepSidePanel
                                isShown={!!selectedStep}
                                namespace={namespace}
                                pipelineName={name}
                                step={step}
                                tab={tab}
                                setTab={setTab}
                                onClose={() => selectStep(null)}
                            />
                        )}
                    </>
                )}
            </>
        </Page>
    );
};

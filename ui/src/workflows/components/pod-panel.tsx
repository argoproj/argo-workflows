import * as React from 'react';
import {useEffect, useState} from 'react';

import {ErrorNotice} from '../../shared/components/error-notice';
import {Notice} from '../../shared/components/notice';
import {ObjectEditor} from '../../shared/components/object-editor';
import {ListWatch} from '../../shared/list-watch';
import {NodePhase, Pod} from '../../shared/models';
import {services} from '../../shared/services';
import {useEditableObject} from '../../shared/use-editable-object';
import {podPanelState} from './pod-panel-state';

function PodViewer({pod}: {pod: Pod}) {
    const {serialization, lang, setLang, resetObject} = useEditableObject<Pod>(pod);
    // useEditableObject only reads its argument once, so every watch update has to be pushed in
    useEffect(() => resetObject(pod), [pod]);
    return <ObjectEditor value={pod} text={serialization} lang={lang} onLangChange={setLang} onChange={null} />;
}

export function PodPanel({namespace, name, podName, nodePhase}: {namespace: string; name: string; podName: string; nodePhase: NodePhase | undefined}) {
    const [pod, setPod] = useState<Pod>();
    const [deleted, setDeleted] = useState(false);
    const [error, setError] = useState<Error>();

    useEffect(() => {
        setPod(undefined);
        setDeleted(false);
        const lw = new ListWatch<Pod>(
            // no list function, so we fake it
            () => Promise.resolve({metadata: {}, items: []}),
            () => services.workflows.watchPod(namespace, name, podName),
            () => setError(null),
            () => setError(null),
            (_, item, type) => {
                // the call right after the (fake) list carries no item
                if (!item) {
                    return;
                }
                setPod(item);
                setDeleted(type === 'DELETED');
            },
            setError
        );
        lw.start();
        return () => lw.stop();
    }, [namespace, name, podName]);

    const state = podPanelState(pod, deleted, nodePhase);

    return (
        <>
            <ErrorNotice error={error} />
            {state === 'waiting' && (
                <Notice>
                    <i className='fa fa-spin fa-circle-notch' /> Waiting for pod {podName} to be created.
                </Notice>
            )}
            {state === 'not-found' && <Notice>Pod {podName} was not found. It may have been deleted or never created.</Notice>}
            {state === 'deleted' && <Notice>Pod {podName} was deleted. Showing its last known state.</Notice>}
            {pod && <PodViewer pod={pod} />}
        </>
    );
}

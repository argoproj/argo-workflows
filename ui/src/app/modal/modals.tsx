import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {ScopedLocalStorage} from '../shared/scoped-local-storage';
import {FirstTimeUser} from './first-time-user';
import {HowAreYouDoing} from './how-are-you-doing';
import {NewVersion} from './new-version';
import {majorMinor} from './version';

export const Modals = ({version}: {version: string}) => {
    const storage = new ScopedLocalStorage('modals');
    const dismissedKey = 'dismissed';
    const lastVersionKey = 'version';
    // when the FTU was dismissed
    const [dismissed, setDismissed] = useState(storage.getItem(dismissedKey, 0));
    useEffect(() => storage.setItem(dismissedKey, dismissed, 0), [dismissed]);
    const [lastVersion, setLastVersion] = useState(storage.getItem(lastVersionKey, 'v0.0.0'));
    useEffect(() => storage.setItem(lastVersionKey, lastVersion, 'v0.0.0'), [lastVersion]);

    const firstTimeUser = dismissed === 0;
    const howAreYouDoing = !firstTimeUser && new Date().getTime() - dismissed > 10000;
    const newVersion = !firstTimeUser && !howAreYouDoing && majorMinor(lastVersion) !== majorMinor(version);

    const shown = firstTimeUser || howAreYouDoing || newVersion;
    const dismiss = () => {
        setDismissed(new Date().getTime());
        if (newVersion) {
            setLastVersion(version);
        }
    };

    return (
        <SlidingPanel hasCloseButton={true} isShown={shown} onClose={dismiss}>
            {firstTimeUser && <FirstTimeUser dismiss={dismiss} />}
            {howAreYouDoing && <HowAreYouDoing dismiss={dismiss} />}
            {newVersion && <NewVersion dismiss={dismiss} version={version} />}
        </SlidingPanel>
    );
};

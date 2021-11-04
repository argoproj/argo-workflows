import * as React from 'react';
import {useEffect, useState} from 'react';
import {ScopedLocalStorage} from '../shared/scoped-local-storage';
import {FirstTimeUserModal} from './first-time-user-modal';
import {NewVersionModal} from './new-version-modal';
import {majorMinor} from './version';

export const ModalSwitch = ({version}: {version: string}) => {
    const storage = new ScopedLocalStorage('modal-switch');

    const firstTimeUserKey = 'firstTimeUser';
    const [firstTimeUser, setFirstTimeUser] = useState(storage.getItem(firstTimeUserKey, true));
    useEffect(() => storage.setItem(firstTimeUserKey, firstTimeUser, true), [firstTimeUser]);

    const lastVersionKey = 'lastVersion';
    const [lastVersion, setLastVersion] = useState(storage.getItem(lastVersionKey, ''));
    useEffect(() => storage.setItem(lastVersionKey, lastVersion, ''), [lastVersion]);

    const xyVersion = majorMinor(version);

    if (firstTimeUser) {
        return <FirstTimeUserModal dismiss={() => setFirstTimeUser(false)} />;
    }
    if (lastVersion !== xyVersion) {
        return <NewVersionModal dismiss={() => setLastVersion(xyVersion)} version={version} />;
    }
    return <></>;
};

import * as React from 'react';
import {useEffect, useState} from 'react';
import {uiUrl} from '../shared/base';
import {FirstTimeUserModal} from './first-time-user-modal';
import {NewVersionModal} from './new-version-modal';
import {majorMinor} from './version';

export const ModalSwitch = ({version}: {version: string}) => {
    // "" === FTU never shown
    // v0.0 === FTU dismissed
    // vx.y === new version for version shown
    const [modal, setModal] = useState<string>(
        (
            decodeURIComponent(document.cookie)
                .split(';')
                .map(x => x.trim())
                .find(x => x.startsWith('modal=')) || ''
        ).replace(/^modal="?(.*?)"?$/, '$1')
    );
    useEffect(() => {
        document.cookie = 'modal=' + modal + ';SameSite=Strict;path=' + uiUrl('');
    }, [modal]);

    const xyVersion = majorMinor(version);

    if (!modal) {
        return <FirstTimeUserModal dismiss={() => setModal('v0.0')} />;
    }
    if (modal !== xyVersion) {
        return <NewVersionModal dismiss={() => setModal(xyVersion)} version={version} />;
    }
    return <></>;
};

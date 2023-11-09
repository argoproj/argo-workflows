import * as React from 'react';
import {useEffect, useState} from 'react';
import {ScopedLocalStorage} from '../shared/scoped-local-storage';
import {FeedbackModal} from './feedback/feedback-modal';
import {FirstTimeUserModal} from './first-time-user/first-time-user-modal';
import {NewVersionModal} from './new-version/new-version-modal';
import {majorMinor} from './version';

export function ModalSwitch({version, modals}: {version: string; modals: {[key: string]: boolean}}) {
    const localStorage = new ScopedLocalStorage('modal');
    const [ftu, setFtu] = useState<string>(localStorage.getItem('ftu', ''));
    useEffect(() => localStorage.setItem('ftu', ftu, ''), [ftu]);

    const now = new Date().getTime();
    const whenToAskForFeedback = now + 6.048e8; // two weeks in milliseconds

    const [feedback, setFeedback] = useState<number>(localStorage.getItem('feedback', null) || whenToAskForFeedback);
    useEffect(() => localStorage.setItem('feedback', feedback, null), [feedback]);

    const [lastVersion, setLastVersion] = useState<string>(localStorage.getItem('version', ''));
    useEffect(() => localStorage.setItem('version', lastVersion, ''), [lastVersion]);

    const majorMinorVersion = majorMinor(version);

    if (modals.firstTimeUser && !ftu) {
        return <FirstTimeUserModal dismiss={() => setFtu('dismissed')} />;
    }
    if (modals.feedback && feedback < now) {
        return <FeedbackModal dismiss={() => setFeedback(whenToAskForFeedback)} />;
    }
    if (modals.newVersion && lastVersion !== majorMinorVersion) {
        return <NewVersionModal dismiss={() => setLastVersion(majorMinorVersion)} version={version} />;
    }
    return <></>;
}

import * as React from 'react';
import {useEffect, useState} from 'react';
import {getCookie, setCookie} from '../shared/cookie';
import {FeedbackModal} from './feedback/feedback-modal';
import {FirstTimeUserModal} from './first-time-user/first-time-user-modal';
import {NewVersionModal} from './new-version/new-version-modal';
import {majorMinor} from './version';

export const ModalSwitch = ({version, modals}: {version: string; modals: {[key: string]: boolean}}) => {
    const ftuCookieName = 'ftu'; // force  using document.cookie='ftu='
    const [ftu, setFtu] = useState(getCookie(ftuCookieName));
    useEffect(() => setCookie(ftuCookieName, String(ftu)), [ftu]);

    const now = new Date().getTime();
    const whenToAskForFeedback = now + 6.048e8; // two weeks in milliseconds

    const feedbackCookieName = 'feedback'; // force by document.cookie='feedback=1' + reload
    const [feedback, setFeedback] = useState<number>(parseFloat(getCookie(feedbackCookieName)) || whenToAskForFeedback);
    useEffect(() => setCookie(feedbackCookieName, String(feedback)), [feedback]);

    const versionCookieName = 'version'; // force by document.cookie='version=' + reload
    const [lastVersion, setLastVersion] = useState<string>(getCookie(versionCookieName));
    useEffect(() => setCookie(versionCookieName, lastVersion), [lastVersion]);

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
};

import * as React from 'react';
import {useEffect, useState} from 'react';

// This wraps the children components in a FTU which allows you to provide an explanation of how to use the feature.
// The child components are hidden behind the panel until the users dismissed the panel by clicking "Continue".
// The fact the users dismissed the panel is persisted in local storage and the panel not show again.
export const FirstTimeUserPanel = ({id, explanation, children, style}: {id: string; explanation: string; children: React.ReactElement; style?: React.CSSProperties}) => {
    const localStorageKey = 'FirstTimeUserPanel/' + id;
    const [dismissed, setDismissed] = useState(localStorage.getItem(localStorageKey) === 'true');

    useEffect(() => {
        localStorage.setItem(localStorageKey, String(dismissed));
    }, [dismissed]);

    if (!dismissed) {
        return (
            <div className='white-box' style={{textAlign: 'center', ...style}}>
                <p>{explanation}</p>
                <p>
                    <a onClick={() => setDismissed(true)}>Continue</a>
                </p>
            </div>
        );
    }
    return <>{children}</>;
};

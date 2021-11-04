import {SlidingPanel} from "argo-ui/src/index";
import * as React from "react";
import {useEffect, useState} from "react";
import {ScopedLocalStorage} from "../shared/scoped-local-storage";
import {AwesomeButton} from "./awesome-button";
import {Icon} from "../shared/components/icon";

export const FirstTimeUser = () => {
    const storage = new ScopedLocalStorage('firstTimeUser');
    const [firstTimeUser, setFirstTimeUser] = useState<boolean>(new Date().getTime() - storage.getItem('ms', 0) > 3000);
    useEffect(() => {
        if (!firstTimeUser) {
            storage.setItem('ms', new Date().getTime(), 0);
        }
    }, [firstTimeUser]);
    return <SlidingPanel hasCloseButton={true} isShown={firstTimeUser} onClose={() => setFirstTimeUser(false)}>
        <div style={{     textAlign: 'center', verticalAlign: 'middle'}}>
        <h3>Tell us what you want to use Argo for - we'll tell you how to do it.</h3>
        <div >
            {Object.entries({
                'brain': 'Machine Learning',
                'sync-alt': 'CI/CD',
                'database': 'Data Processing',
                'table': 'ETL',
                'clock': 'Batch Processing',
                'network-wired': 'Infrastructure Automation',
                'question-circle': 'Something else...'
            }).map(([icon, title]) => (
                <AwesomeButton icon={icon as Icon} title={title} key={icon}/>
            ))}
            <p >
                <a onClick={() => setFirstTimeUser(false)}>Maybe later...</a>
            </p>
        </div>
        </div>
    </SlidingPanel>
}
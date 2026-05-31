import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {CronWorkflowDetails} from './cron-workflow-details';
import {CronWorkflowList} from './cron-workflow-list';

export function CronWorkflowContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<CronWorkflowList />} />
            <Route path=':namespace/:name' element={<CronWorkflowDetails />} />
        </Routes>
    );
}

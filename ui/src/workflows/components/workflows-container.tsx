import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {WorkflowDetails} from './workflow-details/workflow-details';
import {WorkflowsList} from './workflows-list/workflows-list';

export function WorkflowsContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<WorkflowsList />} />
            <Route path=':namespace/:name' element={<WorkflowDetails />} />
        </Routes>
    );
}

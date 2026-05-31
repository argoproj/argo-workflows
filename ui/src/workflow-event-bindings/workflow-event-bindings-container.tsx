import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {WorkflowEventBindings} from './workflow-event-bindings';

export function WorkflowEventBindingsContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<WorkflowEventBindings />} />
        </Routes>
    );
}

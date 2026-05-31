import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {WorkflowTemplateDetails} from './workflow-template-details';
import {WorkflowTemplateList} from './workflow-template-list';

export function WorkflowTemplateContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<WorkflowTemplateList />} />
            <Route path=':namespace/:name' element={<WorkflowTemplateDetails />} />
        </Routes>
    );
}

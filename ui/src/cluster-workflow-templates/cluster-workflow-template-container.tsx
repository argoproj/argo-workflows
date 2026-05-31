import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {ClusterWorkflowTemplateDetails} from './cluster-workflow-template-details';
import {ClusterWorkflowTemplateList} from './cluster-workflow-template-list';

export function ClusterWorkflowTemplateContainer() {
    return (
        <Routes>
            <Route path='' element={<ClusterWorkflowTemplateList />} />
            <Route path=':name' element={<ClusterWorkflowTemplateDetails />} />
        </Routes>
    );
}

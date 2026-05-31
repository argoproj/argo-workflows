import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {ErrorNotice} from '../shared/components/error-notice';
import {WorkflowGraph} from './workflow-graph';
import {WorkflowStatusBadge} from './workflow-status-badge';

export function Widgets() {
    return (
        <Routes>
            <Route path='workflow-graphs/:namespace' element={<WorkflowGraph />} />
            <Route path='workflow-status-badges/:namespace' element={<WorkflowStatusBadge />} />
            <Route path='*' element={<ErrorNotice error={new Error('Widget not found')} />} />
        </Routes>
    );
}

import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {EventFlowPage} from './event-flow-page';

export function EventFlowContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<EventFlowPage />} />
        </Routes>
    );
}

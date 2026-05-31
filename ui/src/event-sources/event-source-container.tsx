import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {EventSourceDetails} from './event-source-details';
import {EventSourceList} from './event-source-list';

export function EventSourceContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<EventSourceList />} />
            <Route path=':namespace/:name' element={<EventSourceDetails />} />
        </Routes>
    );
}

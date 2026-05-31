import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {PluginList} from './plugin-list';

export function PluginsContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<PluginList />} />
        </Routes>
    );
}

import * as React from 'react';
import {Route, Routes} from 'react-router-dom';

import {SensorDetails} from './sensor-details';
import {SensorList} from './sensor-list';

export function SensorsContainer() {
    return (
        <Routes>
            <Route path=':namespace?' element={<SensorList />} />
            <Route path=':namespace/:name' element={<SensorDetails />} />
        </Routes>
    );
}

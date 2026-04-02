import {EventSourceTypes} from '../shared/models/event-source';
import {TriggerTypes} from '../shared/models/sensor';

export const genres = (() => {
    const v: {[label: string]: boolean} = {sensor: true, conditions: true, workflow: true, collapsed: true};
    EventSourceTypes.forEach(x => (v[x] = true));
    TriggerTypes.forEach(x => (v[x] = true));
    return v;
})();

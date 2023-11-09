import {EventSourceTypes} from '../../../../models/event-source';
import {TriggerTypes} from '../../../../models/sensor';

export const genres = (() => {
    const v: {[label: string]: boolean} = {sensor: true, conditions: true, workflow: true, collapsed: true};
    EventSourceTypes.forEach(x => (v[x] = true));
    TriggerTypes.forEach(x => (v[x] = true));
    return v;
})();

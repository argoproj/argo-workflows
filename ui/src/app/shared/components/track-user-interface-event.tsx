import {services} from '../../shared/services';
import {EventParam} from '../services/info-service';

export const TrackEvent = ({name}: {name: string}): null => {
    const param = new Map<EventParam, string>();

    param.set('name', name);
    services.info.collectEvent(param).then();
    return null;
};

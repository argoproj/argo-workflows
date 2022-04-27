import {services} from '../../shared/services';

export const TrackEvent = ({name}: {name: string}): null => {
    services.info.collectEvent(name).then();
    return null;
};

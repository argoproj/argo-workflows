import {useEffect} from 'react';

import {services} from './services';

export function useCollectEvent(name: string) {
    useEffect(() => {
        services.info.collectEvent(name);
    }, []);
}

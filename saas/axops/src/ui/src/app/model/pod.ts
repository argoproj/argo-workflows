export const POD_PHASE = {
    'RUNNING': 'Running'
};

export class Container {
    container_id: string;
    image: string;
    image_id: string;
    last_state: any;
    name: '';
    ready: boolean;
    restart_count: number = 0;
    state: any;
}

export class Pod {
    start_time: number;
    name: string;
    phase: string;
    containers: Container[];

    constructor(data?) {
        if (typeof data === 'object') {
            for (let key in data) {
                if (data.hasOwnProperty(key)) {
                    this[key] = data[key];
                }
            }
        }
    }
}

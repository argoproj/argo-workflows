export class Label {
    id: string;
    key: string;
    value: string;
    type: string;
    reserved: boolean;
    ctime: number;
    // This is a view property
    // @Todo - remove the need for it
    selected: boolean = false;

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

export const LABEL_TYPES = {
    policy: 'policy',
    user: 'user',
    service: 'service'
};

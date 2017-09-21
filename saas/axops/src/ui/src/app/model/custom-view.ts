export class CustomView {
    id: string = '';
    type: string = '';
    user_id: string = '';
    name: string = '';
    info: string = '';
    username: string = '';

    // Ultimately we will start decorating data with annotations
    // For now this piece of code will just extend things.
    constructor(data?) {
        if (typeof data === 'object') {
            for (let key in data) {
                if (data.hasOwnProperty(key) && this.hasOwnProperty(key)) {
                    this[key] = data[key];
                }
            }
        }
    }
}

export const CUSTOM_VIEW_TYPES = {
    testDashboard: 'testDashboard'
};

export class CustomViewInfo {
    repo: string;
    labels: string;
    branch: string;
    template: string;
    template_name?: string;
}

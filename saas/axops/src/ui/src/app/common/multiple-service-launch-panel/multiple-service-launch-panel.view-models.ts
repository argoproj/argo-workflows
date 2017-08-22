export class Session {
    commit: string = '';
    repo: string = '';
    branch: string = '';
}

export class HtmlForm {
    name: number = 0;
    parameters: HtmlParameter[] = [];
}

export class HtmlParameter {
    name: string = '';
    value?: string = '';
}

export const MULTIPLE_SERVICE_LAUNCH_PANEL_TABS = {
    'PARAMETERS': 'parameters',
    'WORKFLOW': 'workflow',
    'OVERVIEW': 'overview',
};

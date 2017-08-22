export interface Project {
    id: string;
    name: string;
    description: string;
    repo: string;
    branch: string;
    categories?: string[];
    labels?: any;
    assets: {
        detail?: string;
        icon: string;
    };
    actions: { [name: string]: ProjectAction };
}

export interface ProjectAction {
    template: string;
    parameters?: any;
}

export const PROMOTED_CATEGORY_NAME = 'promoted';

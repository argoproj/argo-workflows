export const USER_GROUPS = {
    super_admin: 'super_admin',
    admin: 'admin',
    developer: 'developer'
};

export class User {
    auth_schemes: string[] = [];
    first_name: string = '';
    groups: Array<string> = [];
    labels: Array<string> = [];
    id: string = '';
    last_name: string = '';
    password: string = '';
    salt: string = '';
    settings: {} = {};
    state: number = 0;
    username: string = '';
    ctime: number = 0;
    mtime: number = 0;
    view_preferences: any = null;
    anonymous: boolean = false;

    // Ultimately we will start decorating data with annotations
    // For now this piece of code will just extend things.
    constructor(data?) {
        if (data && typeof data === 'object') {
            for (let key in data) {
                if (data.hasOwnProperty(key) && this.hasOwnProperty(key)) {
                    this[key] = data[key];
                }
            }
        }
    }

    // Get user full name
    fullname() {
        let fn = this.first_name ? this.first_name : '';
        fn += ' ';
        fn += this.last_name ? this.last_name : '';
        return fn;
    }

    // Generates an initial for the user
    getInitial() {
        return this.first_name ? this.first_name[0] : this.username[0];
    }

    getName(): string {
        if (this.first_name !== '') {
            return this.fullname();
        } else if (this.username !== '') {
            return this.username;
        } else {
            return '';
        }
    }

    // Returns true if user is admin
    isAdmin() {
        return this.groups.indexOf(USER_GROUPS.admin) > -1 || this.groups.indexOf(USER_GROUPS.super_admin) > -1;
    }

    isSuperAdmin() {
        return this.groups.indexOf(USER_GROUPS.super_admin) > -1;
    }

    giveAdminAccess() {
        if (!this.isAdmin()) {
            this.groups = [USER_GROUPS.admin];
        }
    }

    removeAdminAccess() {
        if (this.isAdmin()) {
            this.groups = [USER_GROUPS.developer];
        }
    }
}


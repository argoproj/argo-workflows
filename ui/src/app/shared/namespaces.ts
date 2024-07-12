const managedNamespaceKey = 'managedNamespace';
const userNamespaceKey = 'userNamespace';
const currentNamespaceKey = 'current_namespace';

export const nsUtils = {
    set userNamespace(value: string) {
        if (value) {
            localStorage.setItem(userNamespaceKey, value);
        } else {
            localStorage.removeItem(userNamespaceKey);
        }
    },

    get userNamespace() {
        return this.fixLocalStorageString(localStorage.getItem(userNamespaceKey));
    },

    set managedNamespace(value: string) {
        if (value) {
            localStorage.setItem(managedNamespaceKey, value);
        } else {
            localStorage.removeItem(managedNamespaceKey);
        }
    },

    get managedNamespace() {
        return this.fixLocalStorageString(localStorage.getItem(managedNamespaceKey));
    },

    fixLocalStorageString(x: string): string {
        // empty string is valid, so we cannot use `truthy`
        if (x == null || x == 'null' || x == 'undefined') {
            return undefined; // explicitly return undefined
        }
        return x;
    },

    // TODO: some of these utils should probably be moved to context
    // eslint-disable-next-line @typescript-eslint/no-unused-vars -- just a temp type, this gets set in app-router
    onNamespaceChange(x: string) {
        // noop
    },

    set currentNamespace(value: string) {
        if (value != null) {
            localStorage.setItem(currentNamespaceKey, value);
        } else {
            localStorage.removeItem(currentNamespaceKey);
        }
        this.onNamespaceChange(this.currentNamespace);
    },

    get currentNamespace() {
        return this.fixLocalStorageString(localStorage.getItem(currentNamespaceKey)) ?? (this.userNamespace || this.managedNamespace);
    },

    // return a namespace, favoring managed namespace when set
    getNamespace(namespace: string) {
        return this.managedNamespace || namespace;
    },

    // return a namespace, never return null/undefined, defaults to "default"
    getNamespaceWithDefault(namespace: string) {
        return namespace || this.currentNamespace || this.userNamespace || this.managedNamespace || 'default';
    }
};

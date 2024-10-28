const managedNamespaceKey = 'managedNamespace';
const userNamespaceKey = 'userNamespace';
const currentNamespaceKey = 'current_namespace';

// just a temp function, this gets set in app-router
let onNamespaceChange = (x: string) => x;
export function setOnNamespaceChange(f: any) {
    onNamespaceChange = f;
}

function fixLocalStorageString(x: string): string {
    // empty string is valid, so we cannot use `truthy`
    if (x == null || x == 'null' || x == 'undefined') {
        return undefined; // explicitly return undefined
    }
    return x;
}

export function setUserNamespace(value: string) {
    if (value) {
        localStorage.setItem(userNamespaceKey, value);
    } else {
        localStorage.removeItem(userNamespaceKey);
    }
}

function getUserNamespace() {
    return fixLocalStorageString(localStorage.getItem(userNamespaceKey));
}

export function setManagedNamespace(value: string) {
    if (value) {
        localStorage.setItem(managedNamespaceKey, value);
    } else {
        localStorage.removeItem(managedNamespaceKey);
    }
}

export function getManagedNamespace() {
    return fixLocalStorageString(localStorage.getItem(managedNamespaceKey));
}

export function setCurrentNamespace(value: string) {
    if (value != null) {
        localStorage.setItem(currentNamespaceKey, value);
    } else {
        localStorage.removeItem(currentNamespaceKey);
    }
    onNamespaceChange(getCurrentNamespace());
}

export function getCurrentNamespace() {
    return fixLocalStorageString(localStorage.getItem(currentNamespaceKey)) ?? (getUserNamespace() || getManagedNamespace());
}

// return a namespace, favoring managed namespace when set
export function getNamespace(namespace: string) {
    return getManagedNamespace() || namespace;
}

// return a namespace, never return null/undefined/empty string, default to "default"
export function getNamespaceWithDefault(namespace: string) {
    return namespace || getCurrentNamespace() || getUserNamespace() || getManagedNamespace() || 'default';
}

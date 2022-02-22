export class ScopedLocalStorage {
    private readonly scope: string;

    constructor(scope: string) {
        this.scope = scope;
    }

    // get the item from storage, returning default if there are any problems
    public getItem<T>(key: string, defaultValue: T) {
        const value = localStorage.getItem(this.scope + '/' + key) || JSON.stringify(defaultValue);
        try {
            const x = JSON.parse(value);
            if (typeof x === typeof defaultValue) {
                return x;
            }
        } catch (ignored) {
            // noop
        }
        return defaultValue;
    }

    // set the item, or clear it if default value
    public setItem<T>(key: string, value: T, defaultValue: T) {
        const x = this.scope + '/' + key;
        const y = JSON.stringify(value);
        if (JSON.stringify(defaultValue) !== y) {
            localStorage.setItem(x, y);
        } else {
            localStorage.removeItem(x);
        }
    }
}

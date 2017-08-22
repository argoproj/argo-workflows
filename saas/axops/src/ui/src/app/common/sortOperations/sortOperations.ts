export class SortOperations {
    static sortBy(collection: any[], key: string, noCaseSensitive?: boolean) {
        return collection.sort((a, b) => {
            let x = a[key];
            let y = b[key];
            if (noCaseSensitive) {
                x = x.toLowerCase();
                y = y.toLowerCase();
            }
            if (x === y) {
                return 0;
            }
            return x < y ? -1 : 1;
        });
    }

    static sortNoCaseSensitive(collection: any[]) {
        return collection.sort(
            (a, b) => {
                return a.toLowerCase().localeCompare(b.toLowerCase());
            }
        );
    }
}

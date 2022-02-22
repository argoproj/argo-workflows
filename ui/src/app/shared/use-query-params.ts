import {History} from 'history';

export function useQueryParams(history: History, set: (p: URLSearchParams) => void): () => void {
    return () => {
        history.listen(newLocation => {
            const newQueryParams = new URLSearchParams(newLocation.search);
            set(newQueryParams);
        });
    };
}

import {History} from 'history';

export function useQueryParams(history: History, set: (p: URLSearchParams) => void): () => void {
    return () => {
        // make sure we return the clean-up func to prevent memory leak warnings
        return history.listen(newLocation => {
            const newQueryParams = new URLSearchParams(newLocation.search);
            set(newQueryParams);
        });
    };
}

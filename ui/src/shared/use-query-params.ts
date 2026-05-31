import {useEffect, useRef} from 'react';
import {useLocation} from 'react-router-dom';

/**
 * Subscribe to changes in the URL query parameters.
 *
 * `set` is invoked with the new `URLSearchParams` whenever `location.search` changes.
 * The initial value is not emitted (components seed their own state from the query
 * params on mount), matching the previous `history.listen`-based behavior.
 */
export function useQueryParams(set: (p: URLSearchParams) => void): void {
    const location = useLocation();
    const isFirstRun = useRef(true);
    useEffect(() => {
        if (isFirstRun.current) {
            isFirstRun.current = false;
            return;
        }
        set(new URLSearchParams(location.search));
        // only re-run when the query string itself changes
    }, [location.search]);
}

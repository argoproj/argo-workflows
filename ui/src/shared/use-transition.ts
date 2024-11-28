import {useEffect, useRef, useState} from 'react';

/**
 * useTransition is a React hook which returns a boolean indicaating whether a transition which takes a fixed amount of
 * time is currently underway. At the time of writing, it is primarily used for syncing JavaScript logic with CSS
 * transitions.
 */
export function useTransition(input: unknown, transitionTime: number) {
    const [transitioning, setTransitioning] = useState(false);
    const prevInput = useRef(input);

    useEffect(() => {
        if (input !== prevInput.current) {
            setTransitioning(true);
            const timeout = setTimeout(() => setTransitioning(false), transitionTime);
            prevInput.current = input;
            return () => clearTimeout(timeout);
        }
    }, [input, transitionTime]);

    return transitioning;
}

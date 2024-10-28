import {MutableRefObject, useCallback, useEffect, useRef, useState} from 'react';

export interface UseResizableWidthArgs {
    /**
     * Whether or not the resizing behavior should be disabled
     */
    disabled?: boolean;
    /**
     * The initial width of the element being resized
     */
    initialWidth?: number;
    /**
     * The maximum width of the element being resized
     */
    maxWidth?: number;
    /**
     * The minimum width of the element being resized
     */
    minWidth?: number;
    /**
     * A ref object which points to the element being resized
     */
    resizedElementRef?: MutableRefObject<HTMLElement>;
}

/**
 * useResizableWidth is a React hook containing the requisite logic for allowing a user to resize an element's width by
 * clicking and dragging another element called the "drag handle". At the time of writing, it is primarily used for
 * resizable side panels.
 */
export function useResizableWidth({disabled, initialWidth, maxWidth, minWidth, resizedElementRef}: UseResizableWidthArgs) {
    const [width, setWidth] = useState(initialWidth ?? 0);

    // The following properties are all maintained as refs (instead of state) because we don't want updating them to
    // trigger a re-render. Usually, only updating the width should trigger a re-render.

    // widthBeforeResize and clientXBeforeResize are needed to calculate the updated width value. they will be set when
    // resizing begins and unset once it ends.
    const widthBeforeResize = useRef(null);
    const clientXBeforeResize = useRef(null);

    // dragOverListener holds the listener for the dragover event which is attached to the document. We need this
    // listener because Firefox has a bug where the event.clientX will always be 0 for drag (not dragover) events.
    const dragOverListener = useRef(null);

    // clientX holds the clientX value from the document.dragover event, which is used in the drag event handler for the
    // "drag handle" element.
    const clientX = useRef(null);

    const handleDragStart = useCallback(
        (event: React.DragEvent<HTMLElement>) => {
            clientXBeforeResize.current = event.clientX;
            widthBeforeResize.current = width;

            function listener(ev: DragEvent) {
                clientX.current = ev.clientX;
            }

            document.addEventListener('dragover', listener);
            dragOverListener.current = listener;
        },
        [width]
    );

    const handleDrag = useCallback(() => {
        if (disabled || clientX.current === null || clientX.current <= 0 || widthBeforeResize.current === null || clientXBeforeResize.current === null) {
            return;
        }

        const newWidth = widthBeforeResize.current + clientXBeforeResize.current - clientX.current;

        if (typeof minWidth === 'number' && newWidth < minWidth) {
            setWidth(minWidth);
        } else if (typeof maxWidth === 'number' && newWidth > maxWidth) {
            setWidth(maxWidth);
        } else {
            setWidth(newWidth);
        }
    }, [disabled, minWidth, maxWidth]);

    const handleDragEnd = useCallback(() => {
        clientXBeforeResize.current = null;
        widthBeforeResize.current = null;
        document.removeEventListener('dragover', dragOverListener.current);
    }, []);

    /**
     * Since the width value is supposed to be the source of truth for the width of the resizable element, we need to
     * make sure to update the width value whenever the width of the resizable element changes for any reason other
     * than the user explicitly resizing it. For instance, if the user shrinks the window and this causes a the
     * resizable element to shrink accordingly, then the width value needs to be updated. If it isn't, then we get
     * weird behavior where the user will start dragging the drag handle, but the resizable element won't resize because
     * the width value maintained by this hook is no longer reflective of reality.
     */
    useEffect(() => {
        if (!resizedElementRef.current || !('ResizeObserver' in window)) {
            return;
        }

        const observer = new ResizeObserver(([element]) => {
            if (disabled) {
                return;
            }

            const observedSize = element.borderBoxSize[0].inlineSize;

            if (observedSize === width) {
                return;
            }

            if (observedSize > maxWidth) {
                setWidth(maxWidth);
            } else if (observedSize < minWidth) {
                setWidth(minWidth);
            } else {
                setWidth(observedSize);
            }
        });

        observer.observe(resizedElementRef.current);

        return () => observer.disconnect();
    }, [disabled, width, minWidth, maxWidth]);

    return {
        width,
        dragHandleProps: {
            draggable: true,
            hidden: disabled,
            onDragStart: handleDragStart,
            onDrag: handleDrag,
            onDragEnd: handleDragEnd
        }
    };
}

import * as React from 'react';
import {useEffect, useRef, useState} from 'react';
import {map} from 'rxjs/operators';
import {Event} from '../../../models';
import {ErrorNotice} from '../../shared/components/error-notice';
import {Notice} from '../../shared/components/notice';
import {Timestamp} from '../../shared/components/timestamp';
import {ToggleButton} from '../../shared/components/toggle-button';
import debounce from '../../shared/debounce';
import {ListWatch} from '../../shared/list-watch';
import {services} from '../../shared/services';

export function EventsPanel({namespace, name, kind}: {namespace: string; name: string; kind: string}) {
    const [showAll, setShowAll] = useState(false);
    const [hideNormal, setHideNormal] = useState(false);
    const [events, setEvents] = useState<Event[]>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        const fieldSelectors: string[] = [];
        if (!showAll) {
            fieldSelectors.push('involvedObject.kind=' + kind);
            fieldSelectors.push('involvedObject.name=' + name);
        }
        if (hideNormal) {
            fieldSelectors.push('type!=Normal');
        }
        const fieldSelector = fieldSelectors.join(',');

        const lw = new ListWatch<Event>(
            // no list function, so we fake it
            () => Promise.resolve({metadata: {}, items: []}),
            () =>
                // ListWatch can only handle Kubernetes Watch Event - so we fake it
                services.workflows.watchEvents(namespace, fieldSelector).pipe(
                    map(
                        x =>
                            x && {
                                type: 'ADDED',
                                object: x
                            }
                    )
                ),
            () => setError(null),
            () => setError(null),
            items => setEvents([...items]),
            setError
        );
        lw.start();
        return () => lw.stop();
    }, [showAll, hideNormal]);

    const tableRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        function calculateTooltips() {
            const table = tableRef.current;

            if (table) {
                const columns = table.querySelectorAll<HTMLDivElement>('.argo-table-list__row .columns');

                // querySelectorAll does not return an array, it returns a NodeList which is not necessarily
                // iterable in older browsers.  As of 2/9/21, we target es5 with typescript, meaning it's
                // not valid to use `for of` loops with NodeList.  When that target is changed or if
                // downlevelIteration is enabled, we can swap this to use a `for of` loop.

                for (let i = 0; i < columns.length; i++) {
                    const col = columns[i];

                    if (col.scrollWidth > col.clientWidth) {
                        col.title = col.textContent;
                    } else {
                        col.title = '';
                    }
                }
            }
        }

        const [debouncedCalculateTooltips, cleanup] = debounce(calculateTooltips, 1000);

        window.addEventListener('resize', debouncedCalculateTooltips);
        calculateTooltips();

        return () => {
            cleanup();
            window.removeEventListener('resize', debouncedCalculateTooltips);
        };
    });

    return (
        <>
            <div style={{margin: 20}}>
                <ToggleButton toggled={showAll} onToggle={() => setShowAll(!showAll)} title='Show all events in the namespace'>
                    Show All
                </ToggleButton>
                <ToggleButton toggled={hideNormal} onToggle={() => setHideNormal(!hideNormal)} title='Hide normal events'>
                    Hide normal
                </ToggleButton>
            </div>
            <ErrorNotice error={error} />
            {!events || events.length === 0 ? (
                <Notice>
                    <i className='fa fa-spin fa-circle-notch' /> Waiting for events. Still waiting for data? Try changing the filters.
                </Notice>
            ) : (
                <div ref={tableRef} className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1'>Type</div>
                        <div className='columns small-2'>Last Seen</div>
                        <div className='columns small-2'>Reason</div>
                        <div className='columns small-2'>Object</div>
                        <div className='columns small-5'>Message</div>
                    </div>
                    {events
                        .filter(e => e && e.lastTimestamp)
                        .sort((a, b) => -a.lastTimestamp.localeCompare(b.lastTimestamp))
                        .map(e => (
                            <div className='row argo-table-list__row' key={e.metadata.uid}>
                                <div className='columns small-1' title={e.type}>
                                    {e.type === 'Normal' ? <i className='fa fa-check-circle status-icon--init' /> : <i className='fa fa-exclamation-circle status-icon--pending' />}
                                </div>
                                <div className='columns small-2'>
                                    <Timestamp date={e.lastTimestamp} />
                                </div>
                                <div className='columns small-2'>{e.reason}</div>
                                <div className='columns small-2'>
                                    {e.involvedObject.kind}/{e.involvedObject.name}
                                </div>
                                <div className='columns small-5'>{e.message}</div>
                            </div>
                        ))}
                </div>
            )}
        </>
    );
}

import {Tooltip} from 'argo-ui/src/components/tooltip/tooltip';
import * as React from 'react';
import {useState} from 'react';

export function ClipboardText({text}: {text: string}) {
    const [justClicked, setJustClicked] = useState(false);

    if (!text) {
        return <></>;
    }

    return (
        <>
            {text}
            &nbsp; &nbsp;
            <Tooltip content={justClicked ? 'Copied!' : 'Copy to clipboard'}>
                <a>
                    <i
                        className={'fa fa-clipboard'}
                        onClick={() => {
                            setJustClicked(true);
                            if (navigator.clipboard && navigator.clipboard.writeText) {
                                navigator.clipboard.writeText(text).catch(err => console.error('Clipboard write failed', err));
                            } else {
                                const textArea = document.createElement('textarea');
                                textArea.value = text;
                                document.body.appendChild(textArea);
                                textArea.select();
                                try {
                                    document.execCommand('copy');
                                    console.log('Fallback: Text copied to clipboard');
                                } catch (err) {
                                    console.error('Fallback: Unable to copy', err);
                                }
                                document.body.removeChild(textArea);
                            }
                            setTimeout(() => setJustClicked(false), 2000);
                        }}
                    />
                </a>
            </Tooltip>
        </>
    );
}

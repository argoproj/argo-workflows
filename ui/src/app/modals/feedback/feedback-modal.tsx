import * as React from 'react';
import {useState} from 'react';
import {BigButton} from '../../shared/components/big-button';
import {Modal} from '../../shared/components/modal/modal';
import {SurveyButton} from '../../shared/components/survey-button';

export function FeedbackModal({dismiss}: {dismiss: () => void}) {
    const [choice, setChoice] = useState(0);
    return (
        <Modal dismiss={dismiss}>
            <h3 style={{textAlign: 'center'}}>How&apos;s it going so far?</h3>
            <div style={{textAlign: 'center'}}>
                <BigButton icon='smile-beam' title='Great' onClick={() => setChoice(1)} />
                <BigButton icon='frown-open' title='Not so good' href='#' onClick={() => setChoice(2)} />
            </div>
            {choice !== 0 && (
                <p style={{textAlign: 'center', paddingTop: 20}}>
                    <SurveyButton />
                </p>
            )}
        </Modal>
    );
}

import * as React from 'react';
import {useState} from 'react';
import {BigButton} from '../shared/components/big-button';
import {Modal} from '../shared/components/modal/modal';
import {SurveyButton} from './survey-button';

export const FeedbackModal = ({dismiss}: {dismiss: () => void}) => {
    const [choice, setChoice] = useState(0);
    return (
        <Modal dismiss={dismiss}>
            <h3 style={{textAlign: 'center'}}>How's it going so far?</h3>
            {!choice ? (
                <div style={{textAlign: 'center'}}>
                    <BigButton icon='smile-beam' title='Great' onClick={() => setChoice(1)} />
                    <BigButton icon='frown-open' title='Not so good' href='#' onClick={() => setChoice(2)} />
                </div>
            ) : (
                <>
                    <p style={{textAlign: 'center'}}>Could you help us improve our product by completing a short survey?</p>
                    <p style={{textAlign: 'center'}}>
                        <SurveyButton />
                    </p>
                </>
            )}
        </Modal>
    );
};

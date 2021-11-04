import * as React from 'react';
import {AwesomeButtons} from './awesome-buttons';
import {Modal} from './modal';

export const HowAreYouDoing = ({dismiss}: {dismiss: () => void}) => (
    <Modal title='How are you doing so far?'>
        <AwesomeButtons
            options={{
                'smile-beam': 'Good',
                'frown-open': 'Not so good'
            }}
            dismiss={dismiss}
        />
    </Modal>
);

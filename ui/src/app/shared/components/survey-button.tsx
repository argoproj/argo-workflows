import * as React from 'react';

export const SurveyButton = () => (
    <>
        <a className='argo-button argo-button--base-o' href='https://forms.gle/LqjDUefjzcC2CKot5?utm_source=argo-ui' target='_blank' rel='noreferrer'>
            Help us by completing a short survey and see the results
        </a>{' '}
        <a href='https://argoproj.github.io/argo-workflows/survey-privacy-policy/?utm_source=argo-ui' style={{color: '#eee'}} target='_blank' rel='noreferrer'>
            <i className='fa fa-question-circle' />
        </a>
    </>
);

import * as React from 'react';

export const SurveyButton = () => (
    <>
        <a className='argo-button argo-button--base-o' href='https://forms.gle/LqjDUefjzcC2CKot5?utm_source=argo-ui' target='_blank'>
            Help us by completing a short survey and see the results
        </a>{' '}
        <a href='https://argo-workflows.readthedocs.io/en/release-3.5/survey-privacy-policy/?utm_source=argo-ui' style={{color: '#eee'}} target='_blank'>
            <i className='fa fa-question-circle' />
        </a>
    </>
);

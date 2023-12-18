import {Page} from 'argo-ui';
import * as React from 'react';

import {uiUrl} from '../../shared/base';
import {useCollectEvent} from '../../shared/components/use-collect-event';

import './help.scss';

export function Help() {
    useCollectEvent('openedHelp');
    return (
        <Page title='Help'>
            <div className='row'>
                <div className='columns large-4 medium-12'>
                    <div className='help-box'>
                        <div className='help-box__ico help-box__ico--manual' />
                        <h3>Documentation</h3>
                        <a href='https://argoproj.github.io/argo-workflows' target='_blank' className='help-box__link' rel='noreferrer'>
                            Online Help
                        </a>
                        <a className='help-box__link' target='_blank' href={uiUrl('apidocs')} rel='noreferrer'>
                            API Docs
                        </a>
                    </div>
                </div>
                <div className='columns large-4 medium-12'>
                    <div className='help-box'>
                        <div className='help-box__ico help-box__ico--email' />
                        <h3>Contact</h3>
                        <a className='help-box__link' target='_blank' href='https://argoproj.github.io/community/join-slack/' rel='noreferrer'>
                            Slack
                        </a>
                    </div>
                </div>
                <div className='columns large-4 medium-12'>
                    <div className='help-box'>
                        <div className='help-box__ico help-box__ico--download' />
                        <h3>Argo CLI</h3>
                        <a className='help-box__link' target='_blank' href='https://github.com/argoproj/argo-workflows/releases/latest' rel='noreferrer'>
                            Releases
                        </a>
                    </div>
                </div>
            </div>
        </Page>
    );
}

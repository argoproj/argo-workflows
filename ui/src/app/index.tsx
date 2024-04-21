import * as React from 'react';
import * as ReactDOM from 'react-dom';
import {App} from './app';

import './i18n';

ReactDOM.render(<App />, document.getElementById('app'));

const mdl = module as any;
if (mdl.hot) {
    mdl.hot.accept('./app.tsx', () => {
        const UpdatedApp = require('./app.tsx').App; // eslint-disable-line @typescript-eslint/no-var-requires
        ReactDOM.render(<UpdatedApp />, document.getElementById('app'));
    });
}

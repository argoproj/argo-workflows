import {createBrowserHistory} from 'history';
import * as React from 'react';
import {createRoot} from 'react-dom/client';

import {App} from './app';

const container = document.getElementById('app');
const root = createRoot(container!);
const history = createBrowserHistory();
root.render(<App history={history} />);

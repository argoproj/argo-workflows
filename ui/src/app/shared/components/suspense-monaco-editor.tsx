import * as React from 'react';
import {MonacoEditorProps} from 'react-monaco-editor';
import MonacoEditor from 'react-monaco-editor';

import {Loading} from './loading';

// lazy load Monaco Editor as it is a gigantic component (which can be split into a separate bundle)
const LazyMonacoEditor = React.lazy(() => {
    return import(/* webpackChunkName: "react-monaco-editor" */ 'react-monaco-editor');
});

// workaround, react-monaco-editor's own default no-op seems to fail when lazy loaded, causing a crash when unmounted
// react-monaco-editor's default no-op: https://github.com/react-monaco-editor/react-monaco-editor/blob/7e5a4938cd328bf95ebc1288967f2037c6023b5a/src/editor.tsx#L184
const noop = () => {}; // tslint:disable-line:no-empty

export const SuspenseMonacoEditor = React.forwardRef(function InnerMonacoEditor(props: MonacoEditorProps, ref: React.MutableRefObject<MonacoEditor>) {
    return (
        <React.Suspense fallback={<Loading />}>
            <LazyMonacoEditor ref={ref} editorWillUnmount={noop} {...props} />
        </React.Suspense>
    );
});

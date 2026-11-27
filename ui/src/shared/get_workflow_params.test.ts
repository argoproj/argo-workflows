import {createBrowserHistory} from 'history';

import {
    getEntrypointFromQuery,
    getEntrypointParametersFromQuery,
    getInitialEntrypoint,
    getInitialEntrypointParameters,
    getLabelsFromQuery,
    getSubmitPrefillSearchParams,
    getWorkflowParametersFromQuery
} from './get_workflow_params';

describe('get_workflow_params', () => {
    it('should return an empty object when there are no query parameters', () => {
        const history = createBrowserHistory();
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({});
    });

    it('should return the parameters provided in the URL', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[key1]=value1&parameters[key2]=value2';
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({
            key1: 'value1',
            key2: 'value2'
        });
    });

    it('should not return any key value pairs which are not in parameters query ', () => {
        const history = createBrowserHistory();
        history.location.search = '?retryparameters[key1]=value1&retryparameters[key2]=value2';
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({});
    });

    it('should only return the parameters provided in the URL', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[key1]=value1&parameters[key2]=value2&test=123';
        const result = getWorkflowParametersFromQuery(history);
        expect(result).toEqual({
            key1: 'value1',
            key2: 'value2'
        });
    });
});

describe('getSubmitPrefillSearchParams', () => {
    it('should return null when there are no prefill parameters', () => {
        expect(getSubmitPrefillSearchParams('?sidePanel=submit&tab=workflows')).toBeNull();
        expect(getSubmitPrefillSearchParams('')).toBeNull();
    });

    it('should keep parameters, entrypoint and labels and drop everything else', () => {
        const result = getSubmitPrefillSearchParams('?sidePanel=submit&tab=workflows&parameters[foo]=bar&entrypoint=my-entry&labels=env%3Dprod&retryparameters[x]=y');
        expect(Array.from(result.entries())).toEqual([
            ['parameters[foo]', 'bar'],
            ['entrypoint', 'my-entry'],
            ['labels', 'env=prod']
        ]);
    });
});

describe('getEntrypointFromQuery', () => {
    it('should return the entrypoint provided in the URL', () => {
        const history = createBrowserHistory();
        history.location.search = '?sidePanel=submit&entrypoint=my-entrypoint';
        expect(getEntrypointFromQuery(history)).toEqual('my-entrypoint');
    });

    it('should return null when the entrypoint is missing or empty', () => {
        const history = createBrowserHistory();
        history.location.search = '?sidePanel=submit';
        expect(getEntrypointFromQuery(history)).toBeNull();
        history.location.search = '?entrypoint=';
        expect(getEntrypointFromQuery(history)).toBeNull();
    });
});

describe('getLabelsFromQuery', () => {
    it('should split comma-separated labels from the URL', () => {
        const history = createBrowserHistory();
        history.location.search = '?labels=key1=value1,key2=value2';
        expect(getLabelsFromQuery(history)).toEqual(['key1=value1', 'key2=value2']);
    });

    it('should decode URL-encoded labels and ignore empty entries', () => {
        const history = createBrowserHistory();
        history.location.search = '?labels=submit-from-ui%3Dtrue,%20custom%3Dvalue%20,,';
        expect(getLabelsFromQuery(history)).toEqual(['submit-from-ui=true', 'custom=value']);
    });

    it('should return the default label when no labels are provided', () => {
        const history = createBrowserHistory();
        history.location.search = '?sidePanel=submit';
        expect(getLabelsFromQuery(history)).toEqual(['submit-from-ui=true']);
        history.location.search = '?labels=,';
        expect(getLabelsFromQuery(history)).toEqual(['submit-from-ui=true']);
    });
});

describe('getEntrypointParametersFromQuery', () => {
    const templateParameters = [
        {name: 'who', value: 'nobody'},
        {name: 'greeting', default: 'hi'}
    ];

    it('should prefill template inputs from the URL and keep the template values otherwise', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[who]=Mason';
        expect(getEntrypointParametersFromQuery(history, templateParameters, [])).toEqual([
            {name: 'who', value: 'Mason'},
            {name: 'greeting', default: 'hi', value: undefined}
        ]);
    });

    it('should keep an explicitly empty value from the URL', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[who]=';
        expect(getEntrypointParametersFromQuery(history, templateParameters, [])[0]).toEqual({name: 'who', value: ''});
    });

    it('should leave out inputs that have the same name as a workflow parameter', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[who]=Mason';
        expect(getEntrypointParametersFromQuery(history, templateParameters, [{name: 'greeting', value: 'hello'}])).toEqual([{name: 'who', value: 'Mason'}]);
    });

    it('should not read inherited properties for inputs named like Object.prototype members', () => {
        const history = createBrowserHistory();
        history.location.search = '?parameters[toString]=';
        const inputs = [
            {name: 'constructor', value: 'c'},
            {name: 'toString', value: 't'}
        ];
        expect(getEntrypointParametersFromQuery(history, inputs, [])).toEqual([
            {name: 'constructor', value: 'c'},
            {name: 'toString', value: ''}
        ]);
    });
});

describe('getInitialEntrypoint', () => {
    const templates = [{name: 'main'}, {name: 'greet'}];

    it('should use the URL entrypoint when the workflow has that template', () => {
        const history = createBrowserHistory();
        history.location.search = '?entrypoint=greet';
        expect(getInitialEntrypoint(history, templates, 'main', '<default>')).toEqual('greet');
    });

    it('should ignore a URL entrypoint that is not a template of the workflow', () => {
        const history = createBrowserHistory();
        history.location.search = '?entrypoint=typo';
        expect(getInitialEntrypoint(history, templates, 'main', '<default>')).toEqual('main');
    });

    it('should use the workflow entrypoint, then the fallback, when the URL sets none', () => {
        const history = createBrowserHistory();
        history.location.search = '';
        expect(getInitialEntrypoint(history, templates, 'main', '<default>')).toEqual('main');
        expect(getInitialEntrypoint(history, templates, undefined, '<default>')).toEqual('<default>');
    });
});

describe('getInitialEntrypointParameters', () => {
    const templates = [{name: 'main'}, {name: 'greet', inputs: {parameters: [{name: 'who', value: 'nobody'}]}}];

    it('should prefill the inputs of the URL entrypoint', () => {
        const history = createBrowserHistory();
        history.location.search = '?entrypoint=greet&parameters[who]=Mason';
        expect(getInitialEntrypointParameters(history, templates, [])).toEqual([{name: 'who', value: 'Mason'}]);
    });

    it('should return no inputs when the URL sets no entrypoint, an unknown one, or one without inputs', () => {
        const history = createBrowserHistory();
        for (const search of ['?parameters[who]=Mason', '?entrypoint=typo', '?entrypoint=main']) {
            history.location.search = search;
            expect(getInitialEntrypointParameters(history, templates, [])).toEqual([]);
        }
    });
});

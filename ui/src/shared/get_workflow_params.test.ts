import {createBrowserHistory} from 'history';

import {getEntrypointFromQuery, getEntrypointParametersFromQuery, getLabelsFromQuery, getSubmitPrefillSearchParams, getWorkflowParametersFromQuery} from './get_workflow_params';

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
});

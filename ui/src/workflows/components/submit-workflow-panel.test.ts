import {createBrowserHistory} from 'history';

describe('SubmitWorkflowPanel URL parsing', () => {
    describe('entrypoint parsing', () => {
        it('should extract entrypoint from URL', () => {
            const history = createBrowserHistory();
            history.location.search = '?sidePanel=submit&entrypoint=my-entrypoint';
            const queryParams = new URLSearchParams(history.location.search);
            const entrypoint = queryParams.get('entrypoint');
            expect(entrypoint).toEqual('my-entrypoint');
        });

        it('should return null when entrypoint is not provided', () => {
            const history = createBrowserHistory();
            history.location.search = '?sidePanel=submit';
            const queryParams = new URLSearchParams(history.location.search);
            const entrypoint = queryParams.get('entrypoint');
            expect(entrypoint).toBeNull();
        });
    });

    describe('labels parsing', () => {
        it('should extract and split labels from URL', () => {
            const history = createBrowserHistory();
            history.location.search = '?sidePanel=submit&labels=key1=value1,key2=value2';
            const queryParams = new URLSearchParams(history.location.search);
            const urlLabels = queryParams.get('labels');
            const labels = urlLabels ? urlLabels.split(',').filter(l => l.trim()) : ['submit-from-ui=true'];
            expect(labels).toEqual(['key1=value1', 'key2=value2']);
        });

        it('should return default label when labels not provided', () => {
            const history = createBrowserHistory();
            history.location.search = '?sidePanel=submit';
            const queryParams = new URLSearchParams(history.location.search);
            const urlLabels = queryParams.get('labels');
            const labels = urlLabels ? urlLabels.split(',').filter(l => l.trim()) : ['submit-from-ui=true'];
            expect(labels).toEqual(['submit-from-ui=true']);
        });

        it('should handle URL-encoded labels', () => {
            const history = createBrowserHistory();
            history.location.search = '?sidePanel=submit&labels=submit-from-ui%3Dtrue,custom%3Dvalue';
            const queryParams = new URLSearchParams(history.location.search);
            const urlLabels = queryParams.get('labels');
            const labels = urlLabels ? urlLabels.split(',').filter(l => l.trim()) : ['submit-from-ui=true'];
            expect(labels).toEqual(['submit-from-ui=true', 'custom=value']);
        });
    });

    describe('combined parameters', () => {
        it('should handle all URL parameters together', () => {
            const history = createBrowserHistory();
            history.location.search = '?sidePanel=submit&entrypoint=retry&parameters[namespace]=development&parameters[action]=retry&labels=submit-from-ui=true';
            const queryParams = new URLSearchParams(history.location.search);

            expect(queryParams.get('sidePanel')).toEqual('submit');
            expect(queryParams.get('entrypoint')).toEqual('retry');
            expect(queryParams.get('labels')).toEqual('submit-from-ui=true');
            const parameters: {[key: string]: string} = {};
            queryParams.forEach((value, key) => {
                const match = key.match(/^parameters\[(.*?)\]$/);
                if (match) {
                    parameters[match[1]] = value;
                }
            });
            expect(parameters).toEqual({
                namespace: 'development',
                action: 'retry'
            });
        });
    });
});

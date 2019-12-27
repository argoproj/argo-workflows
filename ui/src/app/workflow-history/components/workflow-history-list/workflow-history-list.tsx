import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {Observable} from 'rxjs';

import {DataLoader, MockupList, Page, TopBarFilter} from 'argo-ui';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {AppContext, Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';

import {WorkflowListItem} from '..';
import {Query} from '../../../shared/components/query';


export class WorkflowsList extends React.Component<RouteComponentProps<any>> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflows'
                        toolbar={{
                            breadcrumbs: [{title: 'WorkflowHistory', path: uiUrl('workflow-history')}]
                        }}>
                        <div className='workflow-history-list'>
                            TODO
                        </div>
                    </Page>
                )}
            </Consumer>
        );
    }
}

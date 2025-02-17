import {NodeStatus, Workflow} from '../../models';
import {ANNOTATION_KEY_POD_NAME_VERSION} from './annotations';

import {createFNVHash, ensurePodNamePrefixLength, getPodName, getTemplateNameFromNode, k8sNamingHashLength, maxK8sResourceNameLength, POD_NAME_V1, POD_NAME_V2} from './pod-name';

describe('pod names', () => {
    test('createFNVHash', () => {
        expect(createFNVHash('hello')).toEqual(1335831723);
        expect(createFNVHash('world')).toEqual(933488787);
        expect(createFNVHash('You cannot alter your fate. However, you can rise to meet it.')).toEqual(827171719);
    });

    // note: the below is intended to be equivalent to the server-side Go code in workflow/util/pod_name_test.go
    const shortWfName = 'wfname';
    const shortTemplateName = 'templatename';

    const longWfName = 'alongworkflownamethatincludeslotsofdetailsandisessentiallyalargerunonsentencewithpoorstyleandnopunctuationtobehadwhatsoever';
    const longTemplateName =
        'alongtemplatenamethatincludessliightlymoredetailsandiscertainlyalargerunonstnencewithevenworsestylisticconcernsandpreposterouslyeliminatespunctuation';

    test('ensurePodNamePrefixLength', () => {
        let expected = `${shortWfName}-${shortTemplateName}`;
        expect(ensurePodNamePrefixLength(expected)).toEqual(expected);

        expected = `${longWfName}-${longTemplateName}`;
        const actual = ensurePodNamePrefixLength(expected);
        expect(actual.length).toEqual(maxK8sResourceNameLength - k8sNamingHashLength - 1);
    });

    test('getPodName', () => {
        const node = ({
            name: 'nodename',
            id: '1',
            templateName: shortTemplateName
        } as unknown) as NodeStatus;
        const wf = ({
            metadata: {
                name: shortWfName,
                annotations: {
                    [ANNOTATION_KEY_POD_NAME_VERSION]: POD_NAME_V1
                }
            }
        } as unknown) as Workflow;

        const v1podName = node.id;
        const v2podName = `${shortWfName}-${shortTemplateName}-${createFNVHash(node.name)}`;

        expect(getPodName(wf, node)).toEqual(v1podName);
        wf.metadata.annotations[ANNOTATION_KEY_POD_NAME_VERSION] = POD_NAME_V2;
        expect(getPodName(wf, node)).toEqual(v2podName);
        wf.metadata.annotations[ANNOTATION_KEY_POD_NAME_VERSION] = '';
        expect(getPodName(wf, node)).toEqual(v2podName);
        delete wf.metadata.annotations;
        expect(getPodName(wf, node)).toEqual(v2podName);
        expect(getPodName(wf, {...node, name: node.name + '.mycontainername', type: 'Container'})).toEqual(v2podName); // containerSet node check

        wf.metadata.name = longWfName;
        node.templateName = longTemplateName;
        const name = getPodName(wf, node);
        expect(name.length).toEqual(maxK8sResourceNameLength);
    });

    test('getTemplateNameFromNode', () => {
        // case: no template ref or template name
        // expect fallback to empty string
        const node = ({} as unknown) as NodeStatus;
        expect(getTemplateNameFromNode(node)).toEqual('');

        // case: template ref defined but no template name defined
        node.templateRef = {
            name: 'test-template-name',
            template: 'test-template-template'
        };
        expect(getTemplateNameFromNode(node)).toEqual(node.templateRef.template);

        // case: template name defined
        node.templateName = 'test-template';
        expect(getTemplateNameFromNode(node)).toEqual(node.templateName);
    });
});

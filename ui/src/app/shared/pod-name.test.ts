import {createFNVHash, ensurePodNamePrefixLength, getPodName, k8sNamingHashLength, maxK8sResourceNameLength, POD_NAME_V1, POD_NAME_V2} from './pod-name';

describe('pod names', () => {
    test('createFNVHash', () => {
        expect(createFNVHash('hello')).toEqual(1335831723);
        expect(createFNVHash('world')).toEqual(933488787);
        expect(createFNVHash('You cannot alter your fate. However, you can rise to meet it.')).toEqual(827171719);
    });

    const nodeName = 'nodename';
    const nodeID = '1';

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
        expect(getPodName(shortWfName, nodeName, shortTemplateName, nodeID, POD_NAME_V2)).toEqual('wfname-templatename-1454367246');
        expect(getPodName(shortWfName, nodeName, shortTemplateName, nodeID, POD_NAME_V1)).toEqual(nodeID);
        expect(getPodName(shortWfName, nodeName, shortTemplateName, nodeID, '')).toEqual(nodeID);

        const name = getPodName(longWfName, nodeName, longTemplateName, nodeID, POD_NAME_V2);
        expect(name.length).toEqual(maxK8sResourceNameLength);
    });
});

export const ViewUtils = {
  scrollParent(el) {
    const regex = /(auto|scroll)/;

    const ps = $(el).parents();

    for (let i = 0; i < ps.length; i += 1) {
      const node = ps[i];
      const overflow = getComputedStyle(node, null).getPropertyValue('overflow') +
        getComputedStyle(node, null).getPropertyValue('overflow-y') +
        getComputedStyle(node, null).getPropertyValue('overflow-x');
      if (regex.test(overflow)) {
        return node;
      }
    }

    return document.body;
  },

  sanitizeRouteParams(params, ...updatedParams: any[]) {
    params = Object.assign(params, ...updatedParams);
    for (const key of Object.keys(params)) {
      if (params[key] === null || params[key] === undefined) {
        delete params[key];
      }
    }
    return params;
  },

  mapLabelsToList(labelsObject): string[] {
    const labels: string[] = [];
    for (const property in labelsObject) {
      if (labelsObject.hasOwnProperty(property)) {
        labels.push(`${property}: ${labelsObject[property]}`);
      }
    }
    return labels;
  },

  mapToKeyValue(object: {[key: string]: string}): {key: string, value: string}[] {
    const keyValueList = [];
    for (const property in object) {
      if (object.hasOwnProperty(property)) {
        keyValueList.push({key: property, value: object[property]});
      }
    }
    return keyValueList;
  },

  capitalizeFirstLetter(word: string) {
    return word.charAt(0).toUpperCase() + word.slice(1);
  },
};

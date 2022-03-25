const units: {[key: string]: number} = {
    // https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go
    'Ki': Math.pow(2, 10),
    'Mi': Math.pow(2, 20),
    'Gi': Math.pow(2, 30),
    'Ti': Math.pow(2, 40),
    'Pi': Math.pow(2, 50),
    'Ei': Math.pow(2, 60),
    // https://www.businessballs.com/glossaries-and-terminology/metric-prefixes-table/
    'm': 0.001,
    '': 1,
    'k': Math.pow(10, 3),
    'M': Math.pow(10, 6),
    'G': Math.pow(10, 9),
    'T': Math.pow(10, 12),
    'P': Math.pow(10, 15),
    'E': Math.pow(10, 18)
};

export const parseResourceQuantity = (x: string): number => {
    const y = /^([.0-9]+)([^0-9]*)$/.exec(x); // we tolerate a decimal point, just because historically rate was a float, but this probably can be removed some day
    return parseFloat(y[1]) * units[y[2]];
};

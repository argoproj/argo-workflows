import {Metrics} from '../../../models/step';
import {parseResourceQuantity} from '../../shared/resource-quantity';

const prettyNumber = (x: number): number => (x < 1 ? x : Math.round(x));
export const totalRate = (metrics: Metrics, replicas: number): number => {
    const rates = Object.entries(metrics || {})
        // the rate will remain after scale-down, so we must filter out, as it'll be wrong
        .filter(([replica, m]) => parseInt(replica, 10) < replicas);
    return rates.length > 0
        ? prettyNumber(
              rates
                  .map(([, m]) => m)
                  .map(m => parseResourceQuantity(m.rate))
                  .reduce((a, b) => a + b, 0)
          )
        : null;
};

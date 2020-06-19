import {TriggerParameter} from "../../model/sensors";
import * as React from "react";

export const TriggerParametersPanel = ({parameters}: { parameters: TriggerParameter[] }) => {
    return <> {parameters.map(p => (
        <div key={p.dest}>
            {p.src ? p.src.dependencyName + " -> " : ""}
            {p.dest}
        </div>
    ))}</>
}
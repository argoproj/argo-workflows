import * as React from "react";
import { YamlViewer } from "../../../shared/components/yaml/yaml-viewer";
import { Workflow } from "../../../../models";
import * as jsYaml from "js-yaml";

export const WorkflowYamlPanel = (props: { workflow: Workflow }) => <div className='white-box'>
  <div className='white-box__details'>
    <YamlViewer value={jsYaml.dump(props.workflow)}/>
  </div>
</div>;
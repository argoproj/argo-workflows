#!/usr/bin/env python3

import argparse
import json
import subprocess
import tempfile

from subprocess import run

template = '''
<!doctype html>
<html>
<head>
  <title>%s</title>

  <script type="text/javascript" src="http://visjs.org/dist/vis.js"></script>
  <link href="http://visjs.org/dist/vis-network.min.css" rel="stylesheet" type="text/css" />

  <style type="text/css">
    #mynetwork {
      width: 1024px;
      height: 600px;
      border: 1px solid lightgray;
    }
  </style>
</head>
<body>

<p>%s</p>

<div id="mynetwork"></div>

<script type="text/javascript">
  // create an array with nodes
  var nodes = new vis.DataSet(
    %s
  );

  // create an array with edges
  var edges = new vis.DataSet(
    %s
  );

  // create a network
  var container = document.getElementById('mynetwork');
  var data = {
    nodes: nodes,
    edges: edges
  };
  var options = {
    layout: {
      hierarchical: {
        direction: 'LR',
        sortMethod: 'directed', 
      }
    }
  };
  var network = new vis.Network(container, data, options);
</script>


</body>
</html>
'''

def main():
    parser = argparse.ArgumentParser(description='Visualize graph of a workflow')
    parser.add_argument('workflow', type=str, help='workflow name')
    args = parser.parse_args()
    res = run(["kubectl", "get", "workflow", "-o", "json", args.workflow  ], stdout=subprocess.PIPE)
    wf = json.loads(res.stdout.decode("utf-8"))
    nodes = []
    edges = []
    colors = {
      'Pending': 'grey',
      'Running': 'blue',
      'Failed': 'red',
      'Succeeded': 'green',
      'Skipped': 'grey',
      'Error': 'red',
    }
    wf_name = wf['metadata']['name']
    for node_id, node_status in wf['status']['nodes'].items():
        if node_status['name'] == wf_name:
            label = node_status['name']
        else:
            label = node_status['name'].replace(wf_name, "")
        node = {'id': node_id, 'label': label, 'color': colors[node_status['phase']]}
        nodes.append(node)
        if 'children' in node_status:
            for child_id in node_status['children']:
              edge = {'from': node_id, 'to': child_id, 'arrows': 'to'}
              edges.append(edge)
    html = template % (wf_name, wf_name, json.dumps(nodes), json.dumps(edges))
    tmpfile = tempfile.NamedTemporaryFile(suffix='.html', delete=False)
    tmpfile.write(html.encode())
    tmpfile.flush()
    run(["open", tmpfile.name])
    

if __name__ == '__main__':
    main()
<!-- This is an auto-generated file. DO NOT EDIT -->
# python

* Needs: >= v3.3
* Image: python:alpine

This plugin runs trusted Python expressions.

Do not use it to run untrusted Python expressions.

This plugin make attempts to sandbox the expression. It removes built-ins that would allow disk or network access.
The plugin itself is allowed limited CPU and memory, and is, of course, contained.


Install:

    kubectl apply -f python-executor-plugin-configmap.yaml

Uninstall:
	
    kubectl delete cm python-executor-plugin 

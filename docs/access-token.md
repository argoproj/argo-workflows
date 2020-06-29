# Access Token

If you want to automate tasks with the Argo Server API or CLI, you will need an access token. 

Firstly, create a role with minimal permissions. This example role for jenkins only permission to update and list workflows:

```shell script
kubectl create role jenkins --verb=list,update --resource=workflows.argoproj.io 
```

Create a service account for your service:

```shell script
kubectl create sa jenkins
```

Bind the service account to the role (in this case in the `argo` namespace):

```shell script
kubectl create rolebinding jenkins --role=jenkins --serviceaccount=argo:jenkins
```

You now need to get a token:

```shell script
SECRET=$(kubectl -n argo get sa jenkins -o=jsonpath='{.secrets[0].name}')
ARGO_TOKEN=$(kubectl -n argo get secret $SECRET -o=jsonpath='{.data.token}' | base64 --decode)
echo $ARGO_TOKEN
ZXlKaGJHY2lPaUpTVXpJMU5pSXNJbXRwWkNJNkltS...
```

Use that token with the CLI (you need to set `ARGO_SERVER` too):

```shell script
ARGO_SERVER=http://localhost:2746 
argo list
```

Use that token in your API requests, e.g. to list workflows:

```shell script
curl https://localhost:2746/api/v1/workflows/argo -H "Authorisation: Bearer $ARGO_TOKEN"
# 200 OK
```

You should check you cannot do things you're not allowed!

```shell script
curl https://localhost:2746/api/v1/workflow-templates/argo -H "Authorisation: Bearer $ARGO_TOKEN"
# 403 error
```

## Token Revocation

Token compromised?

```shell script
kubectl delete secret $SECRET
```

A new one will be created.

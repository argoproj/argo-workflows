# Access Token

If you want to automate tasks with the Argo Server API or CLI, you will need an access token. 

Firstly, create a role with minimal permissions. This example role for jenkins only permission to update and list workflows:

```sh
kubectl create role jenkins --verb=list,update --resource=workflows.argoproj.io 
```

Create a service account for your service:

```sh
kubectl create sa jenkins
```

Bind the service account to the role (in this case in the `argo` namespace):

```sh
kubectl create rolebinding jenkins --role=jenkins --serviceaccount=argo:jenkins
```

You now need to get a token:

```sh
SECRET=$(kubectl -n argo get sa jenkins -o=jsonpath='{.secrets[0].name}')
ARGO_TOKEN="Bearer $(kubectl -n argo get secret $SECRET -o=jsonpath='{.data.token}' | base64 --decode)"
echo $ARGO_TOKEN
Bearer ZXlKaGJHY2lPaUpTVXpJMU5pSXNJbXRwWkNJNkltS...
```

!!!NOTE
    The `ARGO_TOKEN` should always start with "Bearer ".

Use that token with the CLI (you need to set `ARGO_SERVER` too):

```sh
ARGO_SERVER=http://localhost:2746 
argo list
```

Use that token in your API requests, e.g. to list workflows:

```sh
curl https://localhost:2746/api/v1/workflows/argo -H "Authorisation: $ARGO_TOKEN"
# 200 OK
```

You should check you cannot do things you're not allowed!

```sh
curl https://localhost:2746/api/v1/workflow-templates/argo -H "Authorisation: $ARGO_TOKEN"
# 403 error
```

## Token Revocation

Token compromised?

```sh
kubectl delete secret $SECRET
```

A new one will be created.

# login command

## Procedure

### Step 1. Install dex server to the kind cluster with kong

```
helm install dex -n auth deployments/dex-server --create-namespace
```

> [!NOTE]
> see [llm-operator](https://github.com/llm-operator/llm-operator/tree/main/hack) for the kong installation.

### Step 2. map issuer name

Map the service name to the node IP (or localhost for Docker Desktop), and align it with the issuer URL.

Example /etc/hosts setting for the Docker Desktop:

```
127.0.0.1 kong-kong-proxy.kong
```

## Get id token

Run the following command.

```
go run main.go
```

The login page will then open in your browser.
For registered accounts, see admin fields in `values.yaml`.

## Use id token

```
$ curl --header 'Authorization: Bearer foo' http://localhost/v1/fine_tuning/jobs
{"code":16, "message":"invalid token", "details":[]}

$ export TOKEN=<TOKEN>
$ curl --header "Authorization: Bearer $TOKEN" http://localhost/v1/fine_tuning/jobs
{"object":"list", "data":[], "hasMore":false}
```

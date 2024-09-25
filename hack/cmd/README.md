# login command

## Procedure

Install dex server to the kind cluster with kong

```
helm install dex -n auth deployments/dex-server --create-namespace
```

> [!NOTE]
> see [llmariner](https://github.com/llmariner/llmariner/tree/main/hack) for the kong installation.

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

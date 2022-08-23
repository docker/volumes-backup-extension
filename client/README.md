# docker-credentials-client

This is a client cli to retrieve the Docker credentials in base64.

```shell
$ ./docker-credentials-client get-creds REFERENCE
```

For example, if there is an image with reference `john/my-image:1.0.0` (or equally `docker.io/john/my-image:1.0.0`, you can retrieve the Docker credentials from the `docker.io` (DockerHub) registry running:

```shell
./docker-credentials-client get-creds john/my-image:1.0.0
ey...
```


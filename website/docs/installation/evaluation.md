---
sidebar_position: 1
---

# Evaluation

We provide an [**evaluation** image](https://ghcr.io/ferretdb/ferretdb-eval:2)
for quick testing and experiments.
You'll need [Docker](https://docs.docker.com/get-docker/) installed to run it.

Run this command to start FerretDB with PostgreSQL + DocumentDB extension:

```sh
docker run -d --rm --name ferretdb -p 27017:27017 --platform linux/amd64 ghcr.io/ferretdb/ferretdb-eval:2
```

This command will start a container that includes FerretDB, pre-packaged PostgreSQL with DocumentDB extension, and MongoDB Shell for quick testing and experiments.

:::caution
It is unsuitable for production use cases because it keeps all data inside and loses it on shutdown.
See other installation guides for instructions
that don't have those problems.
:::

With that container running, you can:

- Connect to it with any MongoDB client application using this MongoDB URI:

  ```sh
  mongodb://username:password@127.0.0.1:27017/
  ```

- Or connect to it using MongoDB Shell by just running `mongosh` if you have it installed already.
  If you don't have it installed locally, you can run:

  ```sh
  docker exec -it ferretdb mongosh 'mongodb://username:password@127.0.0.1:27017/'
  ```

- To access PostgreSQL with DocumentDB extension, connect to it by running:

  ```sh
  docker exec -it ferretdb psql -U username postgres
  ```

You can stop the container with:

```sh
docker stop ferretdb
```

In the [next step](usage/concepts.md), we will show you how to perform basic CRUD operations on FerretDB.

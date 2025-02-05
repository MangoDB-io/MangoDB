---
slug: best-alternative-to-mongodb-data-api-and-https-endpoints
title: The Best Alternative for MongoDB Data API and HTTPS Endpoints
authors: [alex]
description: >
  MongoDB's deprecation of Atlas Data API left many developers without a suitable replacement. FerretDB v2 now provides a compatible alternative for the deprecated Atlas Data API.
image: /img/blog/open-source-proprietary-risks.jpg
tags: [open source, sspl, document databases, community]
---

![The Best Alternative for MongoDB Data API and HTTPS Endpoints](/img/blog/open-source-proprietary-risks.jpg)

In September 2024, MongoDB announced the deprecation of Atlas Data API – among other features – leaving many developers affected, and without a suitable replacement.
We are happy to announce that FerretDB v2 now provides a suitable compatible replacement for the deprecated Atlas Data API.

<!--truncate-->

The Data API deprecation affected many developments, especially as it came without sufficient time or alternative migration options.
This is a common theme for many proprietary (or pretend/pseudo open source) projects, and not just MongoDB, you can read more on the topic here.

The good thing is: with the release of FerretDB v2, you can successfully replace Atlas Data API and perform database operations on FerretDB using RESTful endpoints to interact with their databases via HTTP.

In this post, we'll walk you through how to use FerretDB's Data API with Postman, demonstrating how you can query, insert, update, and delete documents – all without needing a MongoDB server.

## Why Data API matters

Some programming environments just don't have a direct way to connect to a database – no native MongoDB driver, no built-in support, nothing.
Without a driver, you'd normally have to wrap your logic inside another application or depend on a separate service just to send or retrieve data.
That's extra work, extra dependencies, and extra points of failure.

A Data API changes that – suddenly, your database is accessible through simple HTTP requests.
A basic curl command can fetch or modify data.
So instead of needing a full-fledged backend just to store something in a database, a workflow can send REST requests to a Data API and log information.

For developers already familiar with REST, Data API is essential.
There's no new learning curve – just familiar HTTP methods (GET, POST, PATCH, DELETE).
With FerretDB's Data API stepping in as an alternative to MongoDB's deprecated service, we are ensuring developers can interact with their data without issues, no matter the stack they're working with.

## FerretDB Data API

FerretDB is an open-source MongoDB-compatible database that runs on PostgreSQL.
With MongoDB's Data API deprecated, we saw the need for a new solution that offers similar functionality while remaining open and accessible.

```mermaid
graph LR;
    A[Client] -->|HTTP/Data API| B[FerretDB Data API];
    B -->|SQL Queries| C[PostgreSQL DB];
    A -->|MongoDB Wire Protocol| D[FerretDB Wire Protocol];
    D -->|SQL Queries| C;
```

FerretDB's Data API provides:

- Seamless HTTP-based database access – No need for drivers; interact via simple REST requests.
- MongoDB-compatible queries – Use familiar JSON-based queries and commands.
- PostgreSQL-backed storage – Leverage the reliability of PostgreSQL with a MongoDB-like API.

## Setting up FerretDB Data API

### Get your FerretDB instance running

Before using the Data API, ensure you have the latest release of FerretDB instance running.
The v2 release requires a PostgreSQL with DocumentDB extension as the backend.

To access Data API on FerretDB, set the environment variable or flag to your desired port (`FERRETDB_LISTEN_DATA_API_ADDR`/`--listen-data-api-addr`)

You can set it up in Docker compose:

```yaml
services:
  postgres:
    image: ghcr.io/ferretdb/postgres-documentdb:16.6-0.100-0
    restart: on-failure
    environment:
      - POSTGRES_USER=username
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=postgres
    volumes:
      - ./data:/var/lib/postgresql/data

  ferretdb:
    image: ghcr.io/ferretdb/ferretdb:2.0.0-rc.1
    restart: on-failure
    ports:
      - 27010:27017
      - 8080:8080
    environment:
      - FERRETDB_POSTGRESQL_URL=postgres://username:password@postgres:5432/postgres
      - FERRETDB_LISTEN_DATA_API_ADDR=:8080
networks:
  default:
    name: ferretdb
```

Once your FerretDB instance is running, the Data API endpoint will be available at `http://localhost:8080`.

### Using FerretDB's Data API for CRUD operations

Since you've enabled FerretDB's Data API (`FERRETDB_LISTEN_DATA_API_ADDR=:8080`), let's write, query, update, and delete a single document using `curl`.

#### Insert a document

Insert a document into a collection called users in a database called `test`.

```sh
curl -X POST http://127.0.0.1:8080/action/insertOne \
  -H "Content-Type: application/json" \
  -u username:password \
  -d '{
        "database": "test",
        "collection": "users",
        "document": {
          "name": "Andrew",
          "email": "andrew@example.com",
          "age": 25
        }
      }'
```

Response:

```json
{ "n": 1.0 }
```

#### Query data

Using a `POST` request, let's retrieve the document we just inserted.

```sh
curl -X POST http://127.0.0.1:8080/action/find \
  -H "Content-Type: application/json" \
  -u username:password \
  -d '{
        "database": "test",
        "collection": "users",
        "filter": { "name": "Andrew" }
      }'
```

Response:

```json
{
  "documents": [
    {
      "_id": { "$oid": "67a2e9b70a5e9467a00918b0" },
      "name": "Andrew",
      "email": "andrew@example.com",
      "age": 25
    }
  ]
}
```

#### Update a document

Let's update Alice's email using a `POST` request:

```sh
curl -X POST http://127.0.0.1:8080/action/updateOne \
  -H "Content-Type: application/json" \
  -u username:password \
  -d '{
        "database": "test",
        "collection": "users",
        "filter": { "name": "Andrew" },
        "update": { "$set": { "email": "andrew.new@example.com" } }
      }'
```

Response:

```json
{ "matchedCount": 1, "modifiedCount": 1 }
```

#### Delete a document

And to complete the CRUD operations, let's delete Alice's document using a `POST` request:

```sh
curl -X POST http://127.0.0.1:8080/action/deleteOne \
  -H "Content-Type: application/json" \
  -u username:password \
  -d '{
        "database": "test",
        "collection": "users",
        "filter": { "name": "Andrew" }
      }'
```

Response:

```json
{ "deletedCount": 1 }
```

## Easily replace MongoDB Data API with FerretDB

With just a few months left before MongoDB's Data API and HTTPS endpoints reach end-of-life, it's crucial to migrate to a suitable alternative.
FerretDB's Data API provides a seamless replacement, allowing developers to interact with their databases via HTTP.
You can easily swap out your MongoDB instance for FerretDB and continue working with your data without any interruptions.
Besides, FerretDB is an open source solution – you can host it on your infrastructure, ensuring control, community support, and long-term stability.

Ready to migrate?
[Check out our migration guide on how you can get started](https://github.com/FerretDB/FerretDB) and start using the Data API today!

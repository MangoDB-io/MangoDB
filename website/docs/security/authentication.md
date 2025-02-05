---
sidebar_position: 1
slug: /security/authentication/ # referenced in FerretDB code in error messages
description: Learn to use authentication mechanisms
---

# Authentication

Authentication is the process of validating a user's identity to prevent database access from users without valid credentials.
It is enabled by default.

FerretDB does not store authentication information (usernames and passwords) itself.
Instead, it relies entirely on PostgreSQL's authentication mechanisms.
All user credentials are stored and managed by PostgreSQL.

:::note
FerretDB only connects to PostgreSQL with DocumentDB extension.
Make sure this is setup correctly.
:::

The following diagram illustrates the authentication process:

```mermaid
sequenceDiagram
   alt Authenticated User
       Client->>FerretDB: Send Credentials
       FerretDB->>PostgreSQL: Forward Credentials
       PostgreSQL-->>FerretDB: Validate and Respond
       FerretDB-->>Client: Authentication Success
       Client->>FerretDB: Query Data
       FerretDB->>PostgreSQL: Forward Query
       PostgreSQL-->>FerretDB: Return Data
       FerretDB-->>Client: Return Data
   else Anonymous User
       Client->>FerretDB: Connect (No Credentials)
       FerretDB-->>Client: Connection Successful
       Client->>FerretDB: Query Data
       FerretDB->>PostgreSQL: Forward Request
       PostgreSQL-->>FerretDB: Denied (No Authentication)
       FerretDB-->>Client: Access Denied
   end
```

As shown in the diagram above, FerretDB forwards the user credentials to PostgreSQL for validation.
PostgreSQL validates the credentials against its user information and sends the result back to FerretDB which then responds to the client.
An anonymous client may still connect to FerretDB without authentication, but they cannot access or perform actions on the database.

To access the database, a client must provide valid user credentials.
These credentials (e.g., username and password) must already exist in PostgreSQL.

At the moment, only `SCRAM-SHA-256` authentication mechanism is supported on the client.

## Create users for authenticated connections

To create users for authenticated connections, there are two options:

- Create users directly from within PostgreSQL
- Use the `createUser` command to create PostgreSQL users from within FerretDB.

### Create users directly with PostgreSQL

You may create new users directly from within PostgreSQL using the `CREATE USER` command.
If you access PostgreSQL through the `psql` command-line tool or any other client, you can create a new user with the following command:

```sql
CREATE USER <newuser> WITH PASSWORD '<newpassword>';"
```

Ensure to update `<newuser>` and `<newpassword>` with your desired values.

Once the user is created, you may connect to FerretDB using the new user credentials:

```sh
mongodb://<newuser>:<newpassword>@127.0.0.1:27017/
```

(See [PostgreSQL documentation for more information on creating users](https://www.postgresql.org/docs/current/sql-createuser.html)).

### Create users with FerretDB

All users created within FerretDB are stored as PostgreSQL users.
You may create new PostgreSQL users directly within FerretDB using the `createUser` command.
Other user management commands are also available.

If you are using the MongoDB Shell to connect to FerretDB, you can create a new user with the following command:

```js
db.createUser({
  user: '<newuser>',
  pwd: '<newpassword>',
  roles: []
})
```

:::info
Authorization is not yet supported.
Set `role` as an empty array (`[]`) when creating a user.
:::

You may then connect to FerretDB using the new user credentials:

```sh
mongodb://<newuser>:<newpassword>@127.0.0.1:27017/
```

## Disable authentication

Since FerretDB relies on PostgreSQL for authentication, disabling authentication essentially means that any user may access your data.

To disable authentication, set the `FERRETDB_AUTH` environment variable to `false` or pass the `--no-auth` flag when starting FerretDB.

Disabling authentication is not recommended for production environments.

## Set up PostgreSQL connection

To connect to PostgreSQL, set the `FERRETDB_POSTGRESQL_URL` environment variable or pass the `--postgresql-url` flag when starting FerretDB.

Ensure that PostgreSQL is set up with the appropriate user credentials, either through direct SQL configuration or any other means.
For example, you may set up PostgreSQL user credentials with Docker Compose.

:::note
FerretDB requires a `postgres` database to be initialized for setting up the connection.
:::

For example, the following Docker compose file initializes the PostgreSQL server with the user credentials and connects FerretDB to PostgreSQL:

```yaml
services:
  postgres:
    image: ghcr.io/ferretdb/postgres-documentdb:16
    environment:
      POSTGRES_USER: <username>
      POSTGRES_PASSWORD: <password>
      POSTGRES_DB: postgres
    volumes:
      - ./data:/var/lib/postgresql/data

  ferretdb:
    image: ghcr.io/ferretdb/ferretdb:2
    restart: on-failure
    ports:
      - 27017:27017
    environment:
      FERRETDB_POSTGRESQL_URL: postgres://<username>:<password>@postgres:5432/postgres

networks:
  default:
    name: ferretdb
```

In this example, FerretDB establishes a connection to PostgreSQL using the specified connection string (`postgres://<username>:<password>@postgres:5432/postgres`).
Replace `<username>` and `<password>` with your desired values.

If authentication is enabled, a valid PostgreSQL user credential must be provided in the connection URI to access the database (e.g., `mongodb://<username>:<password>@@127.0.0.1:27017/`).

:::warning
For local connections, including those configured via Docker Compose, PostgreSQL may use `trust` authentication; even if a `POSTGRES_PASSWORD` is set, any user with access to the PostgreSQL server may connect to the database without a password.
Read more on [trust authentication](https://www.postgresql.org/docs/current/auth-trust.html) to learn how to secure your database.
:::

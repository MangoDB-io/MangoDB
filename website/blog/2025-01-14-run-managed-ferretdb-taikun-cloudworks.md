---
slug: run-managed-ferretdb-taikun-cloudworks
title: 'Run Managed FerretDB instance on Taikun CloudWorks'
authors: [alex]
description: >
  Learn how to deploy a fully managed FerretDB instance in production in any cloud environment using Taikun CloudWorks.
image: /img/blog/ferretdb-taikun2.jpg
tags: [tutorial, open source, cloud]
---

![Run Managed FerretDB instance on Taikun CloudWorks](/img/blog/ferretdb-taikun2.jpg)

Do you want a fully managed, production-ready [FerretDB](https://www.ferretdb.com/) instance – in no time?
Taikun CloudWorks now includes FerretDB as part of its numerous applications.

<!--truncate-->

[Taikun CloudWorks](https://taikun.cloud/taikun-cloudworks/) provides automated features to build, manage, and deploy different Kubernetes clusters and applications at scale – including FerretDB and [PostgreSQL](https://www.postgresql.org/) – on a unified interface across different cloud vendors, including [AWS](https://aws.amazon.com/), [Azure](https://azure.microsoft.com/), [GCP](https://cloud.google.com/), [OpenStack](https://www.openstack.org/), and more.

With a FerretDB instance running on Taikun CloudWorks, you can easily run your MongoDB workloads on a production-ready cluster.

Let's dive into the entire setup and see how you can get a managed FerretDB instance running in minutes.

## Prerequisites

Before you begin, ensure you have the following:

- A Kubernetes cluster configured and running on Taikun CloudWorks.
  If you don't have a Kubernetes cluster set up yet, you can [follow this guide to set it up](https://taikun.cloud/docs/installing-applications/)
- [kubectl](https://kubernetes.io/docs/reference/kubectl/)
- [Helm](https://helm.sh/docs/intro/install/)

## Set up a Kubernetes cluster in Taikun CloudWorks

Start by creating a Taikun project.
It will act as a central management place for the Kubernetes cluster.
Read the following documentation guides to learn how to create a Kubernetes cluster in Taikun:

- [Create a project](https://taikun.cloud/docs/taikun-project-creation/)
- [Create Kubernetes Cluster in Taikun](https://taikun.cloud/docs/creating-kubernetes-cluster/)
- [Download Cluster kubeconfig](https://taikun.cloud/docs/accessing-cluster-kubeconfig/)

With the `kubeconfig` file, you can access the Kubernetes cluster from your local machine.
After downloading the `kubeconfig` file, set the `KUBECONFIG` environment variable to point to the file:

```sh
export KUBECONFIG=/<path>/<to>/<kubeconfig-file>.yaml
```

Then create a namespace for the project:

```sh
kubectl create namespace newferret
```

This namespace will house all the resources for deploying the FerretDB instance.

## Install Percona PostgreSQL Operator

You can deploy a managed FerretDB instance on Taikun CloudWorks using the FerretDB Helm chart.

FerretDB relies on PostgreSQL as the backend database, and the Percona PostgreSQL Operator provides a robust and scalable PostgreSQL solution.
So before installing the chart, ensure the [Percona PostgreSQL Operator](https://github.com/percona/percona-postgresql-operator) is installed and running in your Kubernetes cluster.

You can install the Percona PostgreSQL Operator by following their installation guide [here](https://github.com/percona/percona-postgresql-operator#installation) or by running the following command:

```sh
kubectl apply --server-side -f https://raw.githubusercontent.com/percona/percona-postgresql-operator/v2.3.1/deploy/bundle.yaml -n newferret
```

Check to see if the Percona PostgreSQL Operator is running (the status should be `Running`):

```sh
kubectl get pods -n newferret
```

You should see the Percona PostgreSQL Operator pods running:

```text
NAME                                      READY   STATUS    RESTARTS   AGE
percona-postgresql-operator-59d79f547b-cgz9j   1/1     Running     0          25m
```

## Install FerretDB Helm chart

Now you can install the FerretDB Helm chart.
To add the FerretDB Helm chart repository, run the following command:

```sh
helm repo add ferretdb https://chnyda.github.io/ferretdb-helm
```

Then install the FerretDB Helm chart in the `newferret` namespace:

```sh
helm install mydb --namespace newferret ferretdb/ferretdb
```

It may take a few minutes to install the FerretDB instance.

Once its installed and ready, you should see an output similar to the following:

```text
---------------------------------------------------
Thank you for installing ferretdb.

Your release is named new-ferret in namespace newferret.

To connect to your DB, you could need:

Your password is NOT displayed here from security reasons.
You can find it via:
$ kubectl get secret new-ferret-pgdb-pguser-ferretuser -n newferret -o jsonpath="{.data.password}" | base64 --decode

Your database name: ferretdb
You can find it via:
$ kubectl get secret new-ferret-pgdb-pguser-ferretuser -n newferret -o jsonpath="{.data.dbname}" | base64 --decode

Your username: ferretuser
You can find it via:
$ kubectl get secret new-ferret-pgdb-pguser-ferretuser -n newferret -o jsonpath="{.data.user}" | base64 --decode

Your Java Database Connectivity (JDBC) URI string is NOT displayed here from security reasons (consists password).
You can find it via:
$ kubectl get secret new-ferret-pgdb-pguser-ferretuser -n newferret -o jsonpath="{.data.jdbc-uri}" | base64 --decode

Your PgBouncer URI string is NOT displayed here from security reasons (consists password).
You can find it via:
$ kubectl get secret new-ferret-pgdb-pguser-ferretuser -n newferret -o jsonpath="{.data.pgbouncer-uri}" | base64 --decode
```

As you can see, the FerretDB instance is now running in the `newferret` namespace.
You can access the FerretDB user credentials (`password`, `username`, and `database`) using the commands provided in the output above.

## Connect to the FerretDB instance via `mongosh`

To access the FerretDB instance, you need the user credential and service address for the FerretDB instance.
A typical FerretDB connection string looks like this:

```text
mongodb://<username>:<password>@<host>:27017/<database>?authMechanism=PLAIN
```

Run the following command to get the service address:

```sh
kubectl get svc -n newferret
```

Output:

```text
NAME                        TYPE           CLUSTER-IP      EXTERNAL-IP     PORT(S)           AGE
new-ferret-ferretdb         LoadBalancer   10.233.11.15    185.22.96.145   27017:30772/TCP   18m
new-ferret-pgdb-ha          ClusterIP      10.233.63.103   <none>          5432/TCP          65s
new-ferret-pgdb-ha-config   ClusterIP      None            <none>          <none>            65s
new-ferret-pgdb-pgbouncer   ClusterIP      10.233.33.104   <none>          5432/TCP          65s
new-ferret-pgdb-pods        ClusterIP      None            <none>          <none>            65s
new-ferret-pgdb-primary     ClusterIP      None            <none>          5432/TCP          65s
new-ferret-pgdb-replicas    ClusterIP      10.233.21.71    <none>          5432/TCP          65s
```

The service `new-ferret-ferretdb` is the FerretDB instance and the host address `10.233.11.15` and port is `27017`.

Now that you have the service address, you can connect to the FerretDB instance.

Start a temporary `mongosh` pod to connect to the FerretDB instance:

```sh
kubectl run -it --rm --image=mongo:latest mongo-client -- bash
```

Then, connect to the instance using the following connection string:

```sh
root@mongo-client:/# mongosh 'mongodb://ferretuser:<password>@<host>:27017/ferretdb?authMechanism=PLAIN'
```

The output should look like this:

```text
Current Mongosh Log ID: 6752505068cc7a56ace94969
Connecting to:      mongodb://<credentials>@10.233.11.15:27017/ferretdb?authMechanism=PLAIN&directConnection=true&appName=mongosh+2.3.4
Using MongoDB:      7.0.42
Using Mongosh:      2.3.4
For mongosh info see: https://www.mongodb.com/docs/mongodb-shell/
To help improve our products, anonymous usage data is collected and sent to MongoDB periodically (https://www.mongodb.com/legal/privacy-policy).
You can opt-out by running the disableTelemetry() command.
------
   The server generated these startup warnings when booting
   2024-12-06T01:16:00.872Z: Powered by FerretDB v1.24.0 and PostgreSQL 16.3 - Percona Distribution on x86_64-pc-linux-gnu, compiled by gcc.
   2024-12-06T01:16:00.872Z: Please star us on GitHub: https://github.com/FerretDB/FerretDB.
   2024-12-06T01:16:00.872Z: The telemetry state is undecided.
   2024-12-06T01:16:00.873Z: Read more about FerretDB telemetry and how to opt out at https://beacon.ferretdb.com.
------
```

You are now connected to the FerretDB instance.
In the next section, we'll run some basic MongoDB commands on the FerretDB instance.

### Run commands on Managed FerretDB instance

Start by inserting some documents into the database.
We'll use solar system data to analyze planetary characteristics, such as the number of moons and the size of each planet.

The following command inserts these documents into the `space_data` collection:

```js
db.space_data.insertMany([
  { planet: 'Earth', moons: 1, diameter_km: 12742 },
  { planet: 'Mars', moons: 2, diameter_km: 6779 },
  { planet: 'Jupiter', moons: 79, diameter_km: 139820 }
])
```

Now that the data is in place, let's try out a few analytical queries.

#### Query 1: Total moons

Let's start by answering this question: How many moons are there across the planets in the dataset?
Using the `$group` stage in an aggregation pipeline, we can sum the moons field across all documents:

```js
db.space_data.aggregate([
  {
    $group: {
      _id: null,
      total_moons: { $sum: '$moons' }
    }
  }
])
```

The result shows that the total number of moons is `82`.

```json5
[{ _id: null, total_moons: 82 }]
```

#### Query 2: Planets with more than 1 moon

Next, let's find out which planets have more than one moon.
We'll use the `$match` stage to filter documents based on the condition `moons > 1`:

```js
db.space_data.aggregate([
  {
    $match: { moons: { $gt: 1 } }
  }
])
```

The output lists Mars and Jupiter as the planets with more than one moon.

```json5
[
  { _id: ObjectId('67608c94aea003a29ee94971'), planet: 'Mars', moons: 2, diameter_km: 6779 },
  { _id: ObjectId('67608c94aea003a29ee94972'), planet: 'Jupiter', moons: 79, diameter_km: 139820 }
]
```

#### Query 3: Largest planet by diameter

Finally, let's determine which planet has the largest diameter.
By sorting the documents in descending order of the `diameter_km` field and limiting the result to just one document, we can identify the largest planet:

```js
db.space_data.aggregate([
  {
    $sort: { diameter_km: -1 }
  },
  {
    $limit: 1
  }
])
```

The result as shown below indicates that Jupiter, with a diameter of 139,820 km, is the largest planet in our dataset.

```json5
[
  { _id: ObjectId('67608c94aea003a29ee94972'), planet: 'Jupiter', moons: 79, diameter_km: 139820 }
]
```

You have successfully run some interesting analytical commands on the FerretDB instance.
Go ahead to explore more complex queries and operations.

## Conclusion

In this guide, you learned how to deploy a managed FerretDB instance on Taikun CloudWorks and run some MongoDB commands on a FerretDB instance.
Now you can go ahead to test or migrate your MongoDB workloads on a production-ready cluster with ease on Taikun CloudWorks.

We previously covered how to deploy FerretDB on Kubernetes using Taikun CloudWorks.
You can check out the guide [here](https://blog.ferretdb.io/deploy-ferretdb-kubernetes-taikun-cloudworks/#set-up-a-kubernetes-cluster-in-taikun-cloud).

For a guide on how to migrate your MongoDB workloads to FerretDB, check out the [FerretDB documentation](https://docs.ferretdb.io/migration/).
And should you have any questions or need help, feel free to reach out on any of our community channels on [GitHub](https://docs.ferretdb.io/#community)

// Please do not merge changes in this file.

(function () {
  "use strict";

  // Make 4 test databases.
  db.getSiblingDB('foo').coll.insert({});
  db.getSiblingDB('bar').coll.insert({});
  db.getSiblingDB('buz').coll.insert({});
  db.getSiblingDB('baz').coll.insert({});

  const listDatabasesOut = db.adminCommand({listDatabases: 1});
  const dbList = listDatabasesOut.databases;
  let sizeSum = 0;
  for (let i = 0; i < dbList.length; i++) {
    sizeSum += dbList[i].sizeOnDisk;
  }
  assert.eq(sizeSum, listDatabasesOut.totalSize);

  print('test.js passed!');
})();

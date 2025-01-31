// Please do not merge changes in this file.

(function () {
  "use strict";

  const oplog = db.getSiblingDB('local').oplog.rs;
  
  const t = db.foo;
  t.drop();

  t.insertMany([
    {a: 1},
    {a: 1},
  ]);


  // confirm that there's a single oplog entry with a top-level 'd' op.
  t.deleteOne({a: 1});
  assert.eq(1, oplog.find({ns: 'test.foo', op: 'd'}).itcount());
  jsTestLog(oplog.findOne({ns: 'test.foo', op: 'd'}));

  t.insert({a: 1});

  // will write a single applyOps oplog entry for multiple deletes.
  t.deleteMany({a: 1});
  assert.eq(1, oplog.find({ns: 'test.foo', op: 'd'}).itcount());
  jsTestLog(oplog.findOne({'o.applyOps.ns': 'test.foo'}));

  print('test.js passed!');
})();

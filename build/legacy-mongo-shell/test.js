// Please do not merge changes in this file.

(function () {
  "use strict";

  const t = db.foo;
  t.drop();
  db.bar.drop();

  // works.
  assert.commandWorked(t.runCommand({ping: 1}));
  assert.commandWorked(t.runCommand({ping: 1, apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('ping with Stable API parameters passed');

  assert.commandWorked(t.runCommand({insert: 'foo', documents: [{}]}));
  assert.eq(1, t.count());

  assert.commandWorked(t.runCommand({insert: 'foo', documents: [{}], apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  assert.eq(2, t.count());
  jsTestLog('insert with Stable API parameters passed');

  assert.commandWorked(t.runCommand({count: 'foo', apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('count with Stable API parameters passed');

  // works
  assert.commandWorked(t.runCommand({aggregate: 'test', pipeline: [], cursor: {}, apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('aggregate with Stable API parameters passed');

  // not implemented yet
  assert.commandWorked(t.runCommand({collMod: 'foo', apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('collMod with Stable API parameters passed');

  // works
  assert.commandWorked(t.runCommand({create: 'bar', apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('create with Stable API parameters passed');

  // works
  assert.commandWorked(t.runCommand({createIndexes: 'bar', indexes: [{key: {a: 1}, name: 'a'}], apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('createIndexes with Stable API parameters passed');

  assert.commandWorked(t.runCommand({delete: 'bar', deletes: [{q: {}, limit: 0}], apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('delete with Stable API parameters passed');

  assert.commandWorked(t.runCommand({dropIndexes: 'bar', index: 'a', apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('dropIndexes with Stable API parameters passed');

  assert.commandWorked(t.runCommand({drop: 'bar', apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('drop with Stable API parameters passed');

  assert.commandWorked(t.runCommand({dropDatabase: 1, apiVersion: '1', apiStrict: true, apiDeprecationErrors: true}));
  jsTestLog('dropDatabase with Stable API parameters passed');

  print('test.js passed!');
})();

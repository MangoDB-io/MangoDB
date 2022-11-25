const assert = require('assert');

(function() {
  'use strict';
  const coll = db.issue1539;

  coll.drop();

  docs = [
    {_id: 'double', v: 42.13},
    {_id: 'double-whole', v: 42.0},
    {_id: 'double-zero', v: 0.0},
    {_id: 'double-max', v: Number.MAX_VALUE},
    {_id: 'double-smallest', v: Number.MIN_VALUE},
    {_id: 'doubleBig', v: Number(2 << 60)},
    {_id: 'double-null', v: null},
  ];

  coll.insertMany(docs);

  const expectedErrorCode = 2;

  error = null;
  x = {v: {$size: -1}};
  try {
    coll.findOne(x);
  } catch (e) {
    error = e;
    assert.equal(error.code, expectedErrorCode);
    console.log('test passed');
  }
  if (error == null) {
    console.log('test failed');
  }
})();

// Please do not merge changes in this file.

(function () {
  "use strict";

  const t = db.pull_all;

  t.drop();

  t.insert({'_id': 1, 'scores': [0, 2, 5, 5, 1, 0]});
  t.update({'_id': 1}, {$pullAll: {'scores': [0, 5]}});

  const expected = {'_id': 1, 'scores': [2, 1]};
  assert.eq(expected.scores, t.findOne().scores);

  print('test.js passed!');
})();

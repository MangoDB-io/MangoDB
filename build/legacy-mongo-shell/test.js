// Please do not merge changes in this file.

(function () {
  "use strict";

  const t = db.match_div_zero;
  t.drop();

  // divisor cannot be 0, second argument is the divisor.
  let pipe = {$match: {a: {$mod: [0 /* invalid */, 0]}}};
  pipe = [pipe];

  // FailedToParse occurs when cursor is not present.
  let cmd = {pipeline: pipe, cursor: {batchSize: 0}};
  let res = t.runCommand('aggregate', cmd);

  let code = 2; // BadValue for $mod in a $match pipeline stage.

  let assertThrowsMsg = 'expected the following error code: ' + tojson(code);
  assert.commandFailedWithCode(res, code, assertThrowsMsg);

  pipe = {$project: {a: {$mod: [0 /* invalid */, 0]}}};
  pipe = [pipe];

  cmd = {pipeline: pipe, cursor: {batchSize: 0}};
  res = t.runCommand('aggregate', cmd);

  // eslint-disable-next-line max-len
  code = 16610; // Failed to optimize pipeline :: caused by :: can't $mod by zero"

  assertThrowsMsg = 'expected the following error code: ' + tojson(code);
  assert.commandFailedWithCode(res, code, assertThrowsMsg);

  print('test.js passed!');
})();

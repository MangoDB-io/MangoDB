// Please do not merge changes in this file.

(function () {
  "use strict";

  const t = db.foo;
  t.drop();

  assert.commandWorked(t.stats({scale: 1}));
  // eslint-disable-next-line max-len
  assert.commandWorked(t.stats()); // causes a network error and fails because it uses BSON undefined type 0x06.

  print('test.js passed!');
})();

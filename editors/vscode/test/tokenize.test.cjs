const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { before, after, test } = require('node:test');
const textmate = require('vscode-textmate');
const oniguruma = require('vscode-oniguruma');

let registry;
let grammar;
before(async () => {
  const wasm = fs.readFileSync(require.resolve('vscode-oniguruma/release/onig.wasm'));
  await oniguruma.loadWASM(wasm.buffer.slice(wasm.byteOffset, wasm.byteOffset + wasm.byteLength));
  const raw = JSON.parse(fs.readFileSync(path.join(__dirname, '../syntaxes/tbx.tmLanguage.json'), 'utf8'));
  registry = new textmate.Registry({
    onigLib: Promise.resolve({
      createOnigScanner: patterns => new oniguruma.OnigScanner(patterns),
      createOnigString: value => new oniguruma.OnigString(value),
    }),
    loadGrammar: async scope => scope === 'source.tbx' ? raw : null,
  });
  grammar = await registry.loadGrammar('source.tbx');
});
after(() => registry?.dispose());

function tokenize(lines) {
  let stack = textmate.INITIAL;
  return lines.map(line => {
    const result = grammar.tokenizeLine(line, stack);
    stack = result.ruleStack;
    return result.tokens;
  });
}

function scopesAt(tokens, offset) {
  const token = tokens.find(token => token.startIndex <= offset && offset < token.endIndex);
  assert.ok(token, `No token at ${offset}`);
  return token.scopes;
}

function assertContentAndEnd(line, tokens) {
  assert.ok(scopesAt(tokens, line.indexOf('Hello')).includes('meta.tag.content.tbx'));
  assert.deepEqual(scopesAt(tokens, line.lastIndexOf('end')), ['source.tbx']);
}

test('multiline quoted delimiters remain inside a string', () => {
  const line = 'second ] [ | line"|Hello] end';
  const tokens = tokenize(['[act "first line', line])[1];
  for (const delimiter of [']', '[', '|']) {
    assert.ok(scopesAt(tokens, line.indexOf(delimiter)).includes('string.quoted.double.tbx'));
  }
  assertContentAndEnd(line, tokens);
});

test('quoted state survives blank lines', () => {
  const line = 'second ] line"|Hello] end';
  const tokens = tokenize(['[act "first', '', line])[2];
  assert.ok(scopesAt(tokens, line.indexOf(']')).includes('string.quoted.double.tbx'));
  assertContentAndEnd(line, tokens);
});

test('a backslash at end of line does not escape the next line quote', () => {
  const line = '"|Hello] end';
  const tokens = tokenize(['[act "first' + '\\', line])[1];
  assertContentAndEnd(line, tokens);
});

test('escaped quotes and backslashes preserve parity', () => {
  for (let count = 1; count <= 8; count++) {
    const escaped = count % 2 === 1;
    const line = 'second ' + '\\'.repeat(count) + (escaped ? '" ] still quoted"|Hello] end' : '"|Hello] end');
    const tokens = tokenize(['[act "first', line])[1];
    if (escaped) {
      assert.ok(scopesAt(tokens, line.indexOf(']')).includes('string.quoted.double.tbx'));
    }
    assertContentAndEnd(line, tokens);
  }
});

test('nested multiline strings restore surrounding span scopes', () => {
  for (const outer of ['[slow|', '[slow|[happy|']) {
    const line = 'second ] line"|Hello]' + ']'.repeat(outer === '[slow|' ? 1 : 2) + ' end';
    const tokens = tokenize([outer + '[act "first', line])[1];
    assert.ok(scopesAt(tokens, line.indexOf(']')).includes('string.quoted.double.tbx'));
    assertContentAndEnd(line, tokens);
  }
});

test('quotes in spoken span content are not argument strings', () => {
  const line = '[slow|Hello "quoted speech"] end';
  const tokens = tokenize([line])[0];
  const scopes = scopesAt(tokens, line.indexOf('"'));
  assert.ok(scopes.includes('meta.tag.content.tbx'));
  assert.ok(!scopes.includes('string.quoted.double.tbx'));
  assertContentAndEnd(line, tokens);
});

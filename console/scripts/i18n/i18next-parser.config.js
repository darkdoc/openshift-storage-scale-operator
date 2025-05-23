// eslint-disable-next-line @typescript-eslint/no-var-requires, no-undef
const { CustomJSONLexer } = require('./lexers');
const pkgJson = require('../../package.json');

// eslint-disable-next-line no-undef
module.exports = {
  sort: true,
  createOldCatalogs: false,
  keySeparator: false,
  locales: ['en'],
  namespaceSeparator: '~',
  reactNamespace: false,
  defaultNamespace: `plugin__${pkgJson.name}`,
  useKeysAsDefaultValue: true,

  // see below for more details
  lexers: {
    hbs: ['HandlebarsLexer'],
    handlebars: ['HandlebarsLexer'],

    htm: ['HTMLLexer'],
    html: ['HTMLLexer'],

    mjs: ['JavascriptLexer'],
    js: ['JavascriptLexer'], // if you're writing jsx inside .js files, change this to JsxLexer
    ts: ['JavascriptLexer'],
    jsx: ['JsxLexer'],
    tsx: ['JsxLexer'],
    json: [CustomJSONLexer],

    default: ['JavascriptLexer'],
  },
};

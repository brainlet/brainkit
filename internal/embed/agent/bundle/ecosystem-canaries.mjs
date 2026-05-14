import { request as gaxiosRequest } from 'gaxios';
import nodeFetch, {
  Blob as NodeFetchBlob,
  FormData as NodeFetchFormData,
  Headers as NodeFetchHeaders,
} from 'node-fetch';
import { Readable, Transform, Writable, pipeline as streamPipeline } from 'readable-stream';
import Ajv from 'ajv';
import { parse as parseYAML, stringify as stringifyYAML } from 'yaml';
import matter from 'gray-matter';
import xxhashFactory from 'xxhash-wasm';
import { MongoClient } from 'mongodb';
import { ChromaClient } from 'chromadb';
import { v4 as uuidv4, validate as validateUUID } from 'uuid';
import { LRUCache } from 'lru-cache';
import {
  channel as diagnosticsChannel,
  hasSubscribers as diagnosticsHasSubscribers,
  subscribe as diagnosticsSubscribe,
  unsubscribe as diagnosticsUnsubscribe,
} from 'diagnostics_channel';

function boundary(specifier) {
  try {
    globalThis.require(specifier);
    return { specifier, threw: false };
  } catch (err) {
    return {
      specifier,
      threw: true,
      code: err && err.code || '',
      boundaryCode: err && err.boundaryCode || '',
      boundaryClass: err && err.boundaryClass || '',
      packageName: err && err.packageName || '',
      suggestedOwner: err && err.suggestedOwner || '',
    };
  }
}

function assertBoundary(result, boundaryClass) {
  if (
    !result ||
    !result.threw ||
    result.code !== 'BRAINKIT_UNSUPPORTED_DYNAMIC_REQUIRE' ||
    result.boundaryCode !== 'BRAINKIT_UNSUPPORTED_BOUNDARY' ||
    result.boundaryClass !== boundaryClass
  ) {
    throw new Error('expected ' + result.specifier + ' to throw ' + boundaryClass + ' boundary');
  }
  return result;
}

function ok(id, detail) {
  return { id, ok: true, detail };
}

async function runGaxiosLocal(baseURL) {
  const get = await gaxiosRequest({
    url: baseURL + '/ecosystem/json?client=gaxios',
    method: 'GET',
    headers: { 'x-canary': 'gaxios-get' },
    responseType: 'json',
    fetchImplementation: globalThis.fetch,
    retry: false,
  });
  const post = await gaxiosRequest({
    url: baseURL + '/ecosystem/post',
    method: 'POST',
    data: JSON.stringify({ value: 'gaxios-post' }),
    headers: { 'content-type': 'application/json', 'x-canary': 'gaxios-post' },
    responseType: 'json',
    fetchImplementation: globalThis.fetch,
    retry: false,
  });
  if (get.status !== 200 || get.data.header !== 'gaxios-get') {
    throw new Error('gaxios GET mismatch');
  }
  if (post.status !== 200 || post.data.method !== 'POST' || post.data.body !== '{"value":"gaxios-post"}') {
    throw new Error('gaxios POST mismatch');
  }
  return ok('ecosystem/http-gaxios-local', {
    get: get.data,
    post: post.data,
  });
}

async function runNodeFetchLocal(baseURL) {
  if (typeof nodeFetch !== 'function') {
    throw new TypeError('node-fetch default export type is ' + typeof nodeFetch);
  }
  let get;
  let getJSON;
  try {
    get = await nodeFetch(baseURL + '/ecosystem/json?client=node-fetch', {
      headers: { 'x-canary': 'node-fetch-get' },
    });
  } catch (err) {
    err.message = 'node-fetch GET request: ' + err.message;
    throw err;
  }
  try {
    getJSON = await get.json();
  } catch (err) {
    err.message = 'node-fetch GET json: ' + err.message;
    throw err;
  }

  let post;
  let postJSON;
  try {
    post = await nodeFetch(baseURL + '/ecosystem/post', {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        'x-canary': 'node-fetch-post',
      },
      body: JSON.stringify({ value: 'node-fetch-post' }),
    });
  } catch (err) {
    err.message = 'node-fetch POST request: ' + err.message;
    throw err;
  }
  try {
    postJSON = await post.json();
  } catch (err) {
    err.message = 'node-fetch POST json: ' + err.message;
    throw err;
  }

  if (get.status !== 200 || getJSON.header !== 'node-fetch-get') {
    throw new Error('node-fetch GET mismatch');
  }
  if (post.status !== 200 || postJSON.method !== 'POST' || postJSON.body !== '{"value":"node-fetch-post"}') {
    throw new Error('node-fetch POST mismatch');
  }
  return ok('ecosystem/node-fetch-local', {
    get: getJSON,
    post: postJSON,
    exports: {
      Headers: typeof NodeFetchHeaders,
      Blob: typeof NodeFetchBlob,
      FormData: typeof NodeFetchFormData,
    },
  });
}

async function runReadableStream() {
  const chunks = [];
  const upper = new Transform({
    transform(chunk, _encoding, callback) {
      callback(null, String(chunk).toUpperCase());
    },
  });
  const sink = new Writable({
    write(chunk, _encoding, callback) {
      chunks.push(String(chunk));
      callback();
    },
  });
  await new Promise((resolve, reject) => {
    streamPipeline(Readable.from(['brain', 'kit']), upper, sink, (err) => {
      if (err) reject(err);
      else resolve();
    });
  });
  const output = chunks.join('');
  if (output !== 'BRAINKIT') {
    throw new Error('readable-stream output mismatch: ' + output);
  }
  return ok('ecosystem/readable-stream', { output });
}

async function runAJVCodegen() {
  const ajv = new Ajv({ allErrors: true });
  const validate = ajv.compile({
    type: 'object',
    required: ['name', 'score'],
    properties: {
      name: { type: 'string' },
      score: { type: 'number', minimum: 1 },
    },
    additionalProperties: false,
  });
  const valid = validate({ name: 'brainkit', score: 2 });
  const invalid = validate({ name: 'brainkit', score: 0 });
  if (!valid || invalid || typeof validate !== 'function') {
    throw new Error('ajv validation mismatch');
  }
  return ok('ecosystem/ajv-codegen', {
    valid,
    invalid,
    errors: validate.errors || [],
  });
}

async function runYAMLParser() {
  const parsed = parseYAML('name: brainkit\nitems:\n  - maps\n  - runtime\n');
  const rendered = stringifyYAML({ ok: true, count: parsed.items.length }).trim();
  if (parsed.name !== 'brainkit' || rendered.indexOf('count: 2') < 0) {
    throw new Error('yaml parse/stringify mismatch');
  }
  return ok('ecosystem/yaml-parser', {
    name: parsed.name,
    count: parsed.items.length,
    rendered,
  });
}

async function runFrontmatterParser() {
  const doc = matter('---\ntitle: Brainkit\nkind: canary\n---\nBody text');
  if (doc.data.title !== 'Brainkit' || doc.content.trim() !== 'Body text') {
    throw new Error('gray-matter parse mismatch');
  }
  return ok('ecosystem/frontmatter-parser', {
    title: doc.data.title,
    content: doc.content.trim(),
  });
}

async function runXXHashWASM() {
  const xxhash = await xxhashFactory();
  const h32 = xxhash.h32ToString('brainkit');
  const h64 = xxhash.h64ToString('brainkit');
  if (h32 !== 'debd63d6' || h64 !== '7f7a140612a5edb0') {
    throw new Error('xxhash-wasm output mismatch: ' + h32 + ' ' + h64);
  }
  return ok('ecosystem/xxhash-wasm', { h32, h64 });
}

async function runMongoDBBoundary() {
  const client = new MongoClient('mongodb://localhost:27017', { serverSelectionTimeoutMS: 1 });
  const collectionName = client.db('brainkit').collection('canary').collectionName;
  const boundaries = [
    assertBoundary(boundary('kerberos'), 'optional-native'),
    assertBoundary(boundary('@mongodb-js/zstd'), 'optional-native'),
    assertBoundary(boundary('snappy'), 'optional-native'),
    assertBoundary(boundary('mongodb-client-encryption'), 'optional-native'),
  ];
  await client.close();
  return ok('ecosystem/mongodb-boundary', {
    mongoClient: typeof MongoClient,
    collectionName,
    boundaries,
  });
}

async function runChromaDBBoundary() {
  const client = new ChromaClient({ path: 'http://localhost:8000' });
  const boundaries = [
    assertBoundary(boundary('@chroma-core/default-embed'), 'optional-native'),
    assertBoundary(boundary('fastembed'), 'native-addon'),
  ];
  return ok('ecosystem/chromadb-boundary', {
    chromaClient: typeof ChromaClient,
    clientShape: typeof client,
    boundaries,
  });
}

async function runUUIDConditional() {
  const id = uuidv4();
  if (!validateUUID(id)) {
    throw new Error('uuid validation mismatch: ' + id);
  }
  return ok('ecosystem/uuid-conditional', { uuid: id, valid: true });
}

async function runLRUCacheConditional() {
  const cache = new LRUCache({ max: 2 });
  const name = 'brainkit.ecosystem.lru-cache';
  const ch = diagnosticsChannel(name);
  let seen = 0;
  const subscriber = (message, channelName) => {
    if (channelName === name && message && message.op) seen += 1;
  };
  diagnosticsSubscribe(name, subscriber);
  const subscribed = ch.hasSubscribers && diagnosticsHasSubscribers(name);
  cache.set('a', 1);
  cache.set('b', 2);
  cache.set('c', 3);
  ch.publish({ op: 'set', size: cache.size });
  diagnosticsUnsubscribe(name, subscriber);
  if (cache.get('a') !== undefined || cache.get('b') !== 2 || cache.get('c') !== 3) {
    throw new Error('lru-cache eviction mismatch');
  }
  if (!subscribed || seen !== 1 || ch.hasSubscribers || diagnosticsHasSubscribers(name)) {
    throw new Error('diagnostics_channel adjacency mismatch');
  }
  return ok('ecosystem/lru-cache-conditional', {
    size: cache.size,
    seen,
    subscribed,
  });
}

const runners = {
  'ecosystem/http-gaxios-local': runGaxiosLocal,
  'ecosystem/node-fetch-local': runNodeFetchLocal,
  'ecosystem/readable-stream': runReadableStream,
  'ecosystem/ajv-codegen': runAJVCodegen,
  'ecosystem/yaml-parser': runYAMLParser,
  'ecosystem/frontmatter-parser': runFrontmatterParser,
  'ecosystem/xxhash-wasm': runXXHashWASM,
  'ecosystem/mongodb-boundary': runMongoDBBoundary,
  'ecosystem/chromadb-boundary': runChromaDBBoundary,
  'ecosystem/uuid-conditional': runUUIDConditional,
  'ecosystem/lru-cache-conditional': runLRUCacheConditional,
};

export const ecosystemCanaries = {
  async run(id, options) {
    if (!runners[id]) {
      throw new Error('unknown ecosystem canary: ' + id);
    }
    return runners[id]((options && options.baseURL) || '');
  },
  async runAll(options) {
    const out = [];
    for (const id of Object.keys(runners)) {
      try {
        out.push(await runners[id]((options && options.baseURL) || ''));
      } catch (err) {
        err.message = id + ': ' + (err && err.message || err);
        throw err;
      }
    }
    return out;
  },
};

import assert from 'node:assert/strict';
import { after, before, describe, test } from 'node:test';

import { APP_VERSION, GoServer, request } from './support/server.js';

// Adapted from upstream Nightscout suites:
// api.status.test.js
// api.entries.test.js
// api.treatments.test.js
// api.devicestatus.test.js
// api.root.test.js
// api3.basic.test.js
// api3.generic.workflow.test.js

const server = new GoServer();

before(async () => {
  await server.start();
});

after(async () => {
  await server.stop();
});

describe('Nightscout Compatibility', { concurrency: false }, () => {
  test('status endpoints', async () => {
    let res = await request(server, '/api/status.json');
    assert.equal(res.response.status, 200);
    assert.equal(res.json.apiEnabled, true);
    assert.equal(res.json.careportalEnabled, true);
    assert.ok(res.json.settings.enable.includes('careportal'));

    res = await request(server, '/api/status.txt');
    assert.equal(res.response.status, 200);
    assert.equal(res.text, 'STATUS OK');

    res = await request(server, '/api/status.js');
    assert.equal(res.response.status, 200);
    assert.match(res.text, /^this\.serverSettings =/);

    res = await request(server, '/api/status.html');
    assert.equal(res.response.status, 200);

    res = await request(server, '/api/status.png', { redirect: 'manual' });
    assert.equal(res.response.status, 302);
    assert.equal(res.response.headers.get('location'), 'http://img.shields.io/badge/Nightscout-OK-green.png');
  });

  test('verifyauth semantics', async () => {
    let res = await request(server, '/api/verifyauth');
    assert.equal(res.response.status, 200);
    assert.equal(res.json.message.message, 'UNAUTHORIZED');

    res = await request(server, '/api/verifyauth', { apiSecret: true });
    assert.equal(res.response.status, 200);
    assert.equal(res.json.message.message, 'OK');
  });

  test('versions endpoint', async () => {
    const res = await request(server, '/api/versions');
    assert.equal(res.response.status, 200);
    assert.ok(Array.isArray(res.json));
    assert.ok(res.json.length >= 3);
  });

  test('entries contract', async () => {
    const unique = Date.now();
    let res = await request(server, '/api/v1/entries', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([
        {
          type: 'sgv',
          sgv: '199',
          dateString: '2014-07-20T00:44:15.000-07:00',
          date: unique,
          device: 'dexcom'
        },
        {
          type: 'sgv',
          sgv: '200',
          dateString: '2014-07-20T00:44:15.001-07:00',
          date: unique + 1,
          device: 'dexcom'
        }
      ])
    });
    assert.equal(res.response.status, 200);
    assert.ok(Array.isArray(res.json));
    assert.ok(res.json.length >= 2);

    res = await request(server, `/api/v1/entries.json?find[date][$gte]=${unique}&count=100`);
    assert.equal(res.response.status, 200);
    assert.equal(res.json.length, 2);
    assert.equal(res.json[0].sgv, '200');

    const id = res.json[0]._id;
    res = await request(server, `/api/v1/entries/${id}.json`);
    assert.equal(res.response.status, 200);
    assert.equal(res.json[0]._id, id);

    res = await request(server, '/api/v1/echo/entries/sgv.json?find[sgv][$gte]=199');
    assert.equal(res.response.status, 200);
    assert.equal(res.json.storage, 'entries');

    res = await request(server, '/api/v1/entries/preview.json', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([
        {
          type: 'sgv',
          sgv: '201',
          dateString: '2014-07-20T00:44:15.002-07:00',
          date: unique + 2,
          device: 'dexcom'
        }
      ])
    });
    assert.equal(res.response.status, 201);
    assert.equal(res.json.length, 1);

    res = await request(server, '/api/v1/slice/entries/dateString/sgv/2014-07.json?count=100');
    assert.equal(res.response.status, 200);
    assert.ok(Array.isArray(res.json));

    res = await request(server, '/api/v1/times/echo/2014-07/.*T{00..05}:.json?count=20&find[sgv][$gte]=160');
    assert.equal(res.response.status, 200);
    assert.equal(res.json.pattern.length, 6);

    res = await request(server, `/api/v1/entries.json?find[date][$gte]=${unique}&count=100`, {
      method: 'DELETE',
      apiSecret: true
    });
    assert.equal(res.response.status, 200);
  });

  test('treatments contract', async () => {
    let res = await request(server, '/api/v1/treatments', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        eventType: 'Meal Bolus',
        created_at: '2026-04-10T10:00:00.000+0200',
        carbs: '30',
        insulin: '2.00',
        notes: '<IMG SRC="javascript:alert(\'XSS\');">'
      })
    });
    assert.equal(res.response.status, 200);

    res = await request(server, '/api/v1/treatments.json?find[carbs]=30');
    assert.equal(res.response.status, 200);
    assert.equal(res.json.length, 1);
    assert.equal(res.json[0].notes, '<img>');

    res = await request(server, '/api/v1/treatments?find[carbs]=30', {
      method: 'DELETE',
      apiSecret: true
    });
    assert.equal(res.response.status, 200);
  });

  test('devicestatus contract', async () => {
    let res = await request(server, '/api/v1/devicestatus', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        device: 'xdripjs://rigName',
        xdripjs: { state: 6, stateString: 'OK' },
        created_at: '2018-12-16T01:00:52Z'
      })
    });
    assert.equal(res.response.status, 200);

    res = await request(server, '/api/v1/devicestatus.json?find[created_at][$gte]=2018-12-16&find[created_at][$lte]=2018-12-17');
    assert.equal(res.response.status, 200);
    assert.equal(res.json.length, 1);
    assert.equal(res.json[0].utcOffset, 0);
  });

  test('api v3 version and workflow', async () => {
    let res = await request(server, '/api/v3/version', {
      apiSecret: true
    });
    assert.equal(res.response.status, 200);
    assert.equal(res.json.result.version, APP_VERSION);
    assert.equal(res.json.result.apiVersion, '3.0.4');

    const now = 1760000000000 + Math.floor(Math.random() * 100000);
    res = await request(server, '/api/v3/treatments', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        eventType: 'Correction Bolus',
        insulin: 1,
        date: now,
        app: 'jscompat',
        device: 'go-suite'
      })
    });
    assert.equal(res.response.status, 201);
    const identifier = res.json.identifier;

    res = await request(server, `/api/v3/treatments/${identifier}`, { apiSecret: true });
    assert.equal(res.response.status, 200);
    assert.equal(res.json.result.identifier, identifier);

    res = await request(server, `/api/v3/treatments/${identifier}`, {
      method: 'PATCH',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ carbs: 5, insulin: 0.4 })
    });
    assert.equal(res.response.status, 200);
    assert.equal(res.json.result.carbs, 5);

    res = await request(server, `/api/v3/treatments?identifier$eq=${identifier}&fields=identifier,date,subject`, {
      apiSecret: true
    });
    assert.equal(res.response.status, 200);
    assert.equal(res.json.result.length, 1);
    assert.deepEqual(Object.keys(res.json.result[0]).sort(), ['date', 'identifier', 'subject']);

    res = await request(server, `/api/v3/treatments/history/${now - 1}`, { apiSecret: true });
    assert.equal(res.response.status, 200);
    assert.ok(res.json.result.length >= 1);

    res = await request(server, `/api/v3/treatments/${identifier}`, {
      method: 'DELETE',
      apiSecret: true
    });
    assert.equal(res.response.status, 200);
  });

  test('api v3 security and read parity', async () => {
    let res = await request(server, '/api/v3/test');
    assert.equal(res.response.status, 401);
    assert.equal(res.json.message, 'Missing or bad access token or JWT');

    res = await request(server, '/api/v3/test', {
      headers: { Authorization: 'Bearer invalid_token' }
    });
    assert.equal(res.response.status, 401);
    assert.equal(res.json.message, 'Bad access token or JWT');

    res = await request(server, '/api/v3/devicestatus', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        date: 1760000500000,
        app: 'jscompat',
        device: 'read-parity',
        uploaderBattery: 63
      })
    });
    assert.equal(res.response.status, 201);
    const identifier = res.json.identifier;

    res = await request(server, `/api/v3/devicestatus/${identifier}?fields=date,device,subject`, { apiSecret: true });
    assert.equal(res.response.status, 200);
    assert.deepEqual(Object.keys(res.json.result).sort(), ['date', 'device', 'subject']);

    res = await request(server, `/api/v3/devicestatus/${identifier}?fields=_all`, {
      apiSecret: true,
      headers: { 'If-Modified-Since': new Date(Date.now() + 1000).toUTCString() }
    });
    assert.equal(res.response.status, 304);
    assert.equal(res.text, '');

    res = await request(server, `/api/v3/devicestatus/${identifier}?fields=_all`, { apiSecret: true });
    assert.equal(res.response.status, 200);
    assert.equal(Object.prototype.hasOwnProperty.call(res.json.result, '_id'), false);
    assert.equal(res.json.result.identifier, identifier);

    res = await request(server, `/api/v3/devicestatus/${identifier}`, {
      method: 'DELETE',
      apiSecret: true
    });
    assert.equal(res.response.status, 200);

    res = await request(server, `/api/v3/devicestatus/${identifier}`, { apiSecret: true });
    assert.equal(res.response.status, 410);
  });

  test('api v3 patch metadata and normalized date', async () => {
    let res = await request(server, '/api/v3/treatments', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        date: 1760000600000,
        utcOffset: 120,
        app: 'jscompat',
        device: 'patch-suite',
        eventType: 'Correction Bolus',
        insulin: 0.3
      })
    });
    assert.equal(res.response.status, 201);
    const identifier = res.json.identifier;

    res = await request(server, `/api/v3/treatments/${identifier}`, {
      method: 'PATCH',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ carbs: 10 })
    });
    assert.equal(res.response.status, 200);
    assert.equal(res.json.result.carbs, 10);
    assert.equal(res.json.result.modifiedBy, 'api-secret');

    res = await request(server, `/api/v3/treatments/${identifier}`, {
      method: 'PATCH',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ identifier: 'MODIFIED' })
    });
    assert.equal(res.response.status, 400);
    assert.equal(res.json.message, 'Field identifier cannot be modified by the client');

    res = await request(server, '/api/v3/treatments', {
      method: 'POST',
      apiSecret: true,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        date: '2019-06-10T08:07:08,576+02:00',
        app: 'jscompat',
        device: 'normalized-date',
        eventType: 'Correction Bolus',
        insulin: 0.4
      })
    });
    assert.equal(res.response.status, 201);
    const normalizedId = res.json.identifier;

    res = await request(server, `/api/v3/treatments/${normalizedId}`, { apiSecret: true });
    assert.equal(res.response.status, 200);
    assert.equal(res.json.result.utcOffset, 120);
    assert.equal(res.json.result.date, 1560146828576);
  });
});

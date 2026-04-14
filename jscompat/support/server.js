import { once } from 'node:events';
import { spawn } from 'node:child_process';
import { mkdir, rm } from 'node:fs/promises';
import path from 'node:path';

const API_SECRET = 'this is my long pass phrase';
const API_SECRET_SHA1 = 'b723e97aa97846eb92d5264f084b2823f57c4aa1';
const APP_VERSION = '0.1.0';

export class GoServer {
  constructor(port = 18000 + Math.floor(Math.random() * 1000)) {
    this.port = port;
    this.baseUrl = `http://127.0.0.1:${port}`;
    this.process = null;
    this.binaryPath = path.join(process.cwd(), '.tmp', `glycoview-jscompat-${port}`);
  }

  async start() {
    await mkdir(path.dirname(this.binaryPath), { recursive: true });
    const build = spawn('go', ['build', '-o', this.binaryPath, './cmd/glycoview'], {
      cwd: process.cwd(),
      stdio: ['ignore', 'pipe', 'pipe']
    });
    const [buildCode] = await once(build, 'exit');
    if (buildCode !== 0) {
      throw new Error(`go build failed with code ${buildCode}`);
    }

    this.process = spawn(this.binaryPath, [], {
      cwd: process.cwd(),
      env: {
        ...process.env,
        ADDR: `127.0.0.1:${this.port}`,
        APP_VERSION,
        API_SECRET,
        AUTH_DEFAULT_ROLES: 'readable'
      },
      stdio: ['ignore', 'pipe', 'pipe']
    });

    const deadline = Date.now() + 30000;
    while (Date.now() < deadline) {
      try {
        const response = await fetch(`${this.baseUrl}/api/status.txt`);
        if (response.ok) {
          return;
        }
      } catch {
      }
      await new Promise(resolve => setTimeout(resolve, 200));
    }
    throw new Error('Go server did not become ready');
  }

  async stop() {
    if (!this.process) {
      return;
    }
    this.process.kill('SIGINT');
    await Promise.race([
      once(this.process, 'exit').catch(() => {}),
      new Promise(resolve => setTimeout(resolve, 3000))
    ]);
    if (this.process.exitCode === null) {
      this.process.kill('SIGKILL');
      await once(this.process, 'exit').catch(() => {});
    }
    this.process = null;
    await rm(this.binaryPath, { force: true }).catch(() => {});
  }
}

export async function request(server, path, options = {}) {
  const headers = new Headers(options.headers || {});
  if (options.apiSecret === true) {
    headers.set('api-secret', API_SECRET_SHA1);
  }
  const response = await fetch(`${server.baseUrl}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body,
    redirect: options.redirect || 'follow'
  });
  const text = await response.text();
  let json;
  if ((response.headers.get('content-type') || '').includes('application/json')) {
    json = text ? JSON.parse(text) : undefined;
  }
  return { response, text, json };
}

export { API_SECRET, API_SECRET_SHA1, APP_VERSION };

const { app, BrowserWindow, dialog, shell } = require('electron')
const { spawn } = require('node:child_process')
const fs = require('node:fs')
const http = require('node:http')
const net = require('node:net')
const path = require('node:path')

let backend = null
let backendLog = null
let mainWindow = null
let shuttingDown = false

function backendBinary() {
  const name = process.platform === 'win32' ? 'fai-backend.exe' : 'fai-backend'
  return app.isPackaged
    ? path.join(process.resourcesPath, 'backend', name)
    : path.join(__dirname, '..', 'backend', 'bin', name)
}

function webDirectory() {
  return app.isPackaged
    ? path.join(process.resourcesPath, 'web')
    : path.join(__dirname, '..', 'frontend', 'dist')
}

function freePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.unref()
    server.on('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address()
      server.close(() => resolve(port))
    })
  })
}

function waitForHealth(port, timeoutMs = 20000, exitError = () => null) {
  const deadline = Date.now() + timeoutMs
  return new Promise((resolve, reject) => {
    let settled = false
    const finish = (err) => {
      if (settled) return
      settled = true
      if (err) reject(err)
      else resolve()
    }
    const attempt = () => {
      if (settled) return
      const exited = exitError()
      if (exited) {
        finish(exited)
        return
      }
      if (Date.now() > deadline) {
        finish(new Error('The local backend did not become healthy in time.'))
        return
      }
      const req = http.get(
        { host: '127.0.0.1', port, path: '/healthz', timeout: 1500 },
        (res) => {
          res.resume()
          if (res.statusCode === 200) finish()
          else setTimeout(attempt, 250)
        },
      )
      req.on('timeout', () => req.destroy())
      req.on('error', () => setTimeout(attempt, 250))
    }
    attempt()
  })
}

async function startBackend() {
  const binary = backendBinary()
  const web = webDirectory()
  if (!fs.existsSync(binary)) {
    throw new Error(`Backend binary not found at ${binary}. Build it with "make electron-dev".`)
  }
  if (!fs.existsSync(path.join(web, 'index.html'))) {
    throw new Error(`Frontend build not found at ${web}. Build it with "make electron-dev".`)
  }

  const dataDir = app.getPath('userData')
  const attachmentsDir = path.join(dataDir, 'attachments')
  fs.mkdirSync(attachmentsDir, { recursive: true })

  const port = await freePort()
  backendLog = fs.openSync(path.join(dataDir, 'backend.log'), 'a')
  backend = spawn(binary, [], {
    cwd: dataDir,
    env: {
      ...process.env,
      DATABASE_URL: `sqlite://${path.join(dataDir, 'fai.db')}`,
      DISABLE_AUTH: 'true',
      BIND_ADDR: '127.0.0.1',
      PORT: String(port),
      ATTACHMENTS_DIR: attachmentsDir,
      STATIC_DIR: web,
      PUBLIC_URL: `http://127.0.0.1:${port}`,
    },
    stdio: ['ignore', backendLog, backendLog],
    windowsHide: true,
  })

  let exitError = null
  const fail = (err) => {
    if (shuttingDown) return
    exitError ??= err
    if (!mainWindow) return
    dialog.showErrorBox('fai stopped', err.message)
    app.quit()
  }

  backend.on('error', (err) => {
    backend = null
    fail(err)
  })
  backend.on('exit', () => {
    backend = null
    fail(new Error(`The local backend exited. See ${path.join(dataDir, 'backend.log')} for details.`))
  })

  await waitForHealth(port, 20000, () => exitError)
  return port
}

function stopBackend() {
  shuttingDown = true
  if (backend) {
    backend.kill()
    backend = null
  }
  if (backendLog !== null) {
    try {
      fs.closeSync(backendLog)
    } catch {
      // Already closed.
    }
    backendLog = null
  }
}

async function createWindow() {
  const port = await startBackend()
  const origin = `http://127.0.0.1:${port}`

  mainWindow = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 900,
    minHeight: 600,
    title: 'fai',
    backgroundColor: '#0b0b0c',
    show: false,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  })

  mainWindow.once('ready-to-show', () => mainWindow?.show())
  mainWindow.on('closed', () => {
    mainWindow = null
  })
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (/^(https?:|mailto:)/.test(url)) shell.openExternal(url)
    return { action: 'deny' }
  })
  mainWindow.webContents.on('will-navigate', (event, url) => {
    let sameOrigin = false
    try {
      sameOrigin = new URL(url).origin === origin
    } catch {
      // Not a URL we can compare; treat it as external.
    }
    if (sameOrigin) return
    event.preventDefault()
    if (/^(https?:|mailto:)/.test(url)) shell.openExternal(url)
  })

  await mainWindow.loadURL(`${origin}/`)
}

if (!app.requestSingleInstanceLock()) {
  app.quit()
} else {
  app.on('second-instance', () => {
    if (!mainWindow) return
    if (mainWindow.isMinimized()) mainWindow.restore()
    mainWindow.focus()
  })

  app.on('before-quit', stopBackend)
  app.on('window-all-closed', () => app.quit())

  app
    .whenReady()
    .then(createWindow)
    .catch((err) => {
      dialog.showErrorBox('fai failed to start', err.message)
      app.quit()
    })
}

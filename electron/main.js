const { app, BrowserWindow, dialog, ipcMain, Notification, shell } = require('electron')
const { autoUpdater } = require('electron-updater')
const { spawn } = require('node:child_process')
const fs = require('node:fs')
const http = require('node:http')
const net = require('node:net')
const path = require('node:path')

let backend = null
let backendLog = null
let mainWindow = null
let shuttingDown = false
let updateTimer = null

const updateState = {
  status: app.isPackaged ? 'idle' : 'unsupported',
  currentVersion: app.getVersion(),
  version: null,
  progress: 0,
  error: null,
  autoDownload: false,
}

const updatePrefs = { autoDownload: false }
let updatePrefsLoaded = false

const UPDATE_CHECK_INTERVAL = 6 * 60 * 60 * 1000

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

function updatePrefsPath() {
  return path.join(app.getPath('userData'), 'updates.json')
}

function loadUpdatePrefs() {
  if (updatePrefsLoaded) return
  updatePrefsLoaded = true
  try {
    const parsed = JSON.parse(fs.readFileSync(updatePrefsPath(), 'utf8'))
    if (typeof parsed.autoDownload === 'boolean') updatePrefs.autoDownload = parsed.autoDownload
  } catch {
    // Keep the default: no background downloads.
  }
  updateState.autoDownload = updatePrefs.autoDownload
}

function saveUpdatePrefs() {
  fs.mkdirSync(path.dirname(updatePrefsPath()), { recursive: true })
  fs.writeFileSync(updatePrefsPath(), `${JSON.stringify(updatePrefs, null, 2)}\n`)
}

function setUpdateState(patch) {
  Object.assign(updateState, patch)
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('fai:update-state', { ...updateState })
  }
}

function notifyUpdate(title, body) {
  if (!Notification.isSupported()) return
  const notification = new Notification({ title, body })
  notification.on('click', () => {
    if (!mainWindow) return
    if (mainWindow.isMinimized()) mainWindow.restore()
    mainWindow.show()
    mainWindow.focus()
  })
  notification.show()
}

async function checkForUpdates() {
  if (!app.isPackaged) return updateState
  try {
    await autoUpdater.checkForUpdates()
  } catch (err) {
    setUpdateState({ status: 'error', error: err.message })
  }
  return updateState
}

function configureUpdates() {
  if (!app.isPackaged) return

  loadUpdatePrefs()
  autoUpdater.autoDownload = updatePrefs.autoDownload
  autoUpdater.autoInstallOnAppQuit = true
  setUpdateState({ autoDownload: updatePrefs.autoDownload })

  autoUpdater.on('checking-for-update', () => {
    setUpdateState({ status: 'checking', error: null })
  })
  autoUpdater.on('update-available', (info) => {
    const autoDownload = updatePrefs.autoDownload
    setUpdateState({
      status: autoDownload ? 'downloading' : 'available',
      version: info.version,
      progress: 0,
      error: null,
      autoDownload,
    })
    notifyUpdate(
      'fai update available',
      autoDownload
        ? `Version ${info.version} is downloading in the background.`
        : `Version ${info.version} is available. Open fai to download it.`,
    )
  })
  autoUpdater.on('update-not-available', () => {
    setUpdateState({ status: 'idle', version: null, progress: 0, error: null })
  })
  autoUpdater.on('download-progress', (progress) => {
    setUpdateState({ status: 'downloading', progress: progress.percent })
  })
  autoUpdater.on('update-downloaded', (info) => {
    setUpdateState({ status: 'downloaded', version: info.version, progress: 100, error: null })
    notifyUpdate('fai update ready', `Version ${info.version} is ready. Restart fai to install it.`)
  })
  autoUpdater.on('error', (err) => {
    setUpdateState({ status: 'error', error: err.message })
  })

  setTimeout(checkForUpdates, 5000)
  updateTimer = setInterval(checkForUpdates, UPDATE_CHECK_INTERVAL)
  updateTimer.unref()
}

ipcMain.on('fai:app-version', (event) => {
  event.returnValue = app.getVersion()
})

ipcMain.handle('fai:update-state', () => ({ ...updateState }))

ipcMain.handle('fai:update-check', () => checkForUpdates())

ipcMain.handle('fai:update-download', async () => {
  if (updateState.status !== 'available') return { ...updateState }
  try {
    await autoUpdater.downloadUpdate()
  } catch (err) {
    setUpdateState({ status: 'error', error: err.message })
  }
  return { ...updateState }
})

ipcMain.handle('fai:update-auto-download', (_event, enabled) => {
  updatePrefs.autoDownload = Boolean(enabled)
  try {
    saveUpdatePrefs()
  } catch {
    // Keep the in-memory preference if the file cannot be written.
  }
  autoUpdater.autoDownload = updatePrefs.autoDownload
  setUpdateState({ autoDownload: updatePrefs.autoDownload })
  if (updatePrefs.autoDownload && updateState.status === 'available') {
    autoUpdater.downloadUpdate().catch((err) => {
      setUpdateState({ status: 'error', error: err.message })
    })
  }
  return { ...updateState }
})

ipcMain.handle('fai:update-install', () => {
  if (updateState.status !== 'downloaded') return false
  autoUpdater.quitAndInstall()
  return true
})

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
      preload: path.join(__dirname, 'preload.js'),
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

  app.on('before-quit', () => {
    if (updateTimer) {
      clearInterval(updateTimer)
      updateTimer = null
    }
    stopBackend()
  })
  app.on('window-all-closed', () => app.quit())

  app
    .whenReady()
    .then(async () => {
      await createWindow()
      configureUpdates()
    })
    .catch((err) => {
      dialog.showErrorBox('fai failed to start', err.message)
      app.quit()
    })
}

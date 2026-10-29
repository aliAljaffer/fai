const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('fai', {
  version: ipcRenderer.sendSync('fai:app-version'),
  getUpdateState: () => ipcRenderer.invoke('fai:update-state'),
  checkForUpdates: () => ipcRenderer.invoke('fai:update-check'),
  downloadUpdate: () => ipcRenderer.invoke('fai:update-download'),
  setAutoDownload: (enabled) => ipcRenderer.invoke('fai:update-auto-download', enabled),
  installUpdate: () => ipcRenderer.invoke('fai:update-install'),
  onUpdateState: (callback) => {
    const listener = (_event, state) => callback(state)
    ipcRenderer.on('fai:update-state', listener)
    return () => ipcRenderer.removeListener('fai:update-state', listener)
  },
})

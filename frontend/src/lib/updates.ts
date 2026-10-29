import { useSyncExternalStore } from 'react'

export type UpdateStatus =
  | 'unsupported'
  | 'idle'
  | 'checking'
  | 'available'
  | 'downloading'
  | 'downloaded'
  | 'error'

export interface UpdateState {
  status: UpdateStatus
  currentVersion: string
  version: string | null
  progress: number
  error: string | null
  autoDownload: boolean
}

interface FaiDesktop {
  version: string
  getUpdateState: () => Promise<UpdateState>
  checkForUpdates: () => Promise<UpdateState>
  downloadUpdate: () => Promise<UpdateState>
  setAutoDownload: (enabled: boolean) => Promise<UpdateState>
  installUpdate: () => Promise<boolean>
  onUpdateState: (callback: (state: UpdateState) => void) => () => void
}

declare global {
  interface Window {
    fai?: FaiDesktop
  }
}

const buildVersion = (import.meta.env.VITE_APP_VERSION || 'dev').replace(/^v/, '')

let state: UpdateState = {
  status: window.fai ? 'idle' : 'unsupported',
  currentVersion: window.fai?.version ?? buildVersion,
  version: null,
  progress: 0,
  error: null,
  autoDownload: false,
}

const listeners = new Set<() => void>()
let initialized = false

function emit() {
  for (const listener of listeners) listener()
}

function setState(next: UpdateState) {
  state = next
  emit()
}

function init() {
  if (initialized) return
  initialized = true
  const bridge = window.fai
  if (!bridge) return
  bridge.onUpdateState(setState)
  bridge.getUpdateState().then(setState).catch(() => {})
}

function subscribe(listener: () => void) {
  init()
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

function getSnapshot() {
  return state
}

export function useUpdateState() {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot)
}

export function appVersion() {
  return window.fai?.version ?? buildVersion
}

export function isDesktop() {
  return Boolean(window.fai)
}

export async function checkForUpdates() {
  await window.fai?.checkForUpdates()
}

export async function downloadUpdate() {
  await window.fai?.downloadUpdate()
}

export async function setAutoDownload(enabled: boolean) {
  await window.fai?.setAutoDownload(enabled)
}

export async function installUpdate() {
  await window.fai?.installUpdate()
}

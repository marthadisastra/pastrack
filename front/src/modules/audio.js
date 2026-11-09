import { settingsState, isModuleActive } from './state'
let audioContext
export async function unlockAudio() {
  const Context = window.AudioContext || window.webkitAudioContext
  if (!Context) throw new Error('Audio preview is unavailable in this browser.')
  audioContext ||= new Context()
  if (audioContext.state === 'suspended') await audioContext.resume()
  return audioContext
}
export async function playTone(tone, volume = 60, gain = 100) {
  const context = await unlockAudio(),
    now = context.currentTime
  const frequencies = {
    klaxon: [660, 880, 660],
    pulse: [900, 900, 900],
    chime: [660, 880],
    thud: [180, 120],
    beep: [520],
  }
  const notes = frequencies[tone] || frequencies.beep
  notes.forEach((frequency, index) => {
    const oscillator = context.createOscillator(),
      level = context.createGain(),
      at = now + index * 0.16
    oscillator.type = 'sine'
    oscillator.frequency.value = frequency
    level.gain.setValueAtTime(0, at)
    level.gain.linearRampToValueAtTime(
      Math.min(1, Math.max(0, volume / 100)) * Math.min(1, Math.max(0, gain / 100)) * 0.08,
      at + 0.015,
    )
    level.gain.exponentialRampToValueAtTime(0.0001, at + 0.13)
    oscillator.connect(level)
    level.connect(context.destination)
    oscillator.start(at)
    oscillator.stop(at + 0.15)
    oscillator.onended = () => {
      oscillator.disconnect()
      level.disconnect()
    }
  })
}
export function notifyAudio(notification) {
  const audio = settingsState.value.audio
  if (!audio.enabled || audioContext?.state !== 'running') return
  const critical = notification.severity === 'critical',
    event = audio.events.find(
      (event) => event.id === (notification.audioEvent || (critical ? 'siphon' : 'ble')),
    )
  if (!event?.enabled || (event.id === 'impact' && !isModuleActive('ext.queclink.kinematics')))
    return
  const time = new Date().toLocaleTimeString('en-GB', {
    timeZone: 'Asia/Jakarta',
    hour: '2-digit',
    minute: '2-digit',
  })
  const quiet =
    audio.quiet &&
    (audio.quietFrom < audio.quietTo
      ? time >= audio.quietFrom && time < audio.quietTo
      : time >= audio.quietFrom || time < audio.quietTo)
  if (quiet && !(critical && audio.neverMuteCritical)) return
  playTone(event.tone, audio.volume, event.gain).catch(() => {})
}

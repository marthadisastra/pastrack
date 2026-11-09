export function mergeFuelSamples(points, samples, vehicleId, meta, limit = 5000) {
  if (!meta || !vehicleId) return { points, to: meta?.to }
  const from = new Date(meta.from).getTime()
  let latest = new Date(meta.to).getTime()
  const byID = new Map(points.map((point) => [String(point.sampleId), point]))
  for (const sample of samples) {
    const at = new Date(sample.at).getTime()
    if (String(sample.vehicleId) !== String(vehicleId) || !Number.isFinite(at) || at < from)
      continue
    byID.set(String(sample.sampleId), sample)
    latest = Math.max(latest, at)
  }
  const merged = [...byID.values()].sort((a, b) => new Date(a.at) - new Date(b.at))
  return {
    points: merged.slice(Math.max(0, merged.length - limit)),
    to: new Date(latest).toISOString(),
  }
}

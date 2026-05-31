import type { Job } from '../types/job'
import { API_BASE } from './config'

export async function fetchJobs(): Promise<Job[]> {
  const res = await fetch(`${API_BASE}/jobs`, { credentials: 'include' })
  if (!res.ok) throw new Error(`Failed to fetch jobs: ${res.status}`)
  return res.json()
}

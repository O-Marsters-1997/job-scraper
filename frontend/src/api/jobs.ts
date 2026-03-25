import type { Job } from '../types/job'

export async function fetchJobs(): Promise<Job[]> {
  const res = await fetch('http://localhost:8080/jobs')
  if (!res.ok) throw new Error(`Failed to fetch jobs: ${res.status}`)
  return res.json()
}

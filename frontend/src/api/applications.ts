import type { Application, ApplicationWithDetails, JobApplicationSummary } from '../types/application'

import { API_BASE } from './config'

export async function fetchApplications(statusId?: string): Promise<ApplicationWithDetails[]> {
  const url = statusId
    ? `${API_BASE}/applications?status_id=${statusId}`
    : `${API_BASE}/applications`
  const res = await fetch(url, { credentials: 'include' })
  if (!res.ok) throw new Error(`Failed to fetch applications: ${res.status}`)
  return res.json()
}

export async function createApplication(data: {
  job_id: string
  status_id?: string
  notes?: string
  applied_at?: string | null
  salary_info?: string
}): Promise<Application> {
  const res = await fetch(`${API_BASE}/applications`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (res.status === 409) throw new Error('application_exists')
  if (!res.ok) throw new Error(`Failed to create application: ${res.status}`)
  return res.json()
}

export async function updateApplication(
  id: string,
  data: {
    status_id?: string
    notes?: string
    applied_at?: string | null
    salary_info?: string
  },
): Promise<Application> {
  const res = await fetch(`${API_BASE}/applications/${id}`, {
    method: 'PATCH',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(`Failed to update application: ${res.status}`)
  return res.json()
}

export async function deleteApplication(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/applications/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  })
  if (!res.ok) throw new Error(`Failed to delete application: ${res.status}`)
}

export async function fetchApplicationsForJobs(
  jobIds: string[],
): Promise<Record<string, JobApplicationSummary>> {
  if (jobIds.length === 0) return {}
  const res = await fetch(
    `${API_BASE}/applications/for-jobs?job_ids=${jobIds.join(',')}`,
    { credentials: 'include' },
  )
  if (!res.ok) throw new Error(`Failed to fetch applications for jobs: ${res.status}`)
  return res.json()
}

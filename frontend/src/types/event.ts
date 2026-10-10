export type EventType = "job_opened" | "job_dismissed" | "alert_opened";

export interface EventInput {
	type: EventType;
	subject_id: string;
}

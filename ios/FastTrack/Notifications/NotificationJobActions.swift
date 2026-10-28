import SwiftUI

private struct NotificationJobActions: ViewModifier {
    @Binding var job: Job?
    let model: NotificationsModel
    @State private var isShowing = false
    @State private var reasonsJob: Job?
    @State private var noteJob: Job?

    func body(content: Content) -> some View {
        content
            .onChange(of: job) { isShowing = job != nil }
            .onChange(of: isShowing) { if !isShowing { job = nil } }
            .confirmationDialog(
                job?.title ?? "Job actions", isPresented: $isShowing, titleVisibility: .visible, presenting: job
            ) { job in
                JobActionsMenu(
                    job: job, onDecide: { act(job, $0) },
                    onDismissWithReason: { reasonsJob = job }, onAddNote: { noteJob = job })
            }
            .sheet(item: $reasonsJob) { job in
                DismissReasonSheet { act(job, .dismiss(reasons: $0)) }
            }
            .sheet(item: $noteJob) { job in
                NoteSheet(load: { try await model.note(for: job) }) { act(job, .note($0)) }
            }
    }

    private func act(_ job: Job, _ action: TriageAction) {
        Task { await model.perform(action, on: job) }
    }
}

extension View {
    func notificationJobActions(job: Binding<Job?>, model: NotificationsModel) -> some View {
        modifier(NotificationJobActions(job: job, model: model))
    }
}
